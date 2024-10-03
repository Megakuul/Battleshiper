import 'SHIMS';
import { Server } from 'SERVER';
import { manifest, prerendered } from 'MANIFEST';

const server = new Server(manifest);

/**
 * @param {import('aws-lambda').APIGatewayProxyEventV2} event
 * @param {import('aws-lambda').Context} context
 * @returns {Promise<import('aws-lambda').APIGatewayProxyResultV2>}
 */
export const handler = async (event, context) => {
  await server.init({ env: process.env });

  try {
    const method = event.requestContext.http.method;
    const { headers, body } = event;

    headers.origin = process.env.ORIGIN ?? headers.origin ?? `https://${event.requestContext.domainName}`;
    const url = new URL(`${headers.origin}${event.rawPath}`);
    url.search = event.rawQueryString;

    // Warning: This is specific to battleshiper because the proxy (edge router) is trusted.
    // If no server explicitly sets the x-forwarded-host header, this poses a security risk (csrf).
    if (headers["x-forwarded-host"]) {
      headers.host = headers["x-forwarded-host"];
    }

    /** @type {Request} */
    const req = new Request(url, {
      method: method,
      headers: new Headers(event.headers),
      body: body,
    })
  
    /** @type {Response} */
    const res = await server.respond(req, {
      platform: { context },
      getClientAddress: (request) => {
        return headers["x-forwarded-for"] || event.requestContext.http.sourceIp;
      }
    })

    return {
      statusCode: res.status,
      headers: Object.fromEntries(res.headers.entries()),
      body: await res.text(),
      isBase64Encoded: false,
    }
  } catch (err) {
    return {
      statusCode: 500,
      headers: {
        "Content-Type": "application/json"
      },
      body: {
        message: `internal server error`,
        error: DEBUGMODE ? err.message : undefined,
      },
      isBase64Encoded: false,
    }
  }
};