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

  const url = new URL(`${event.requestContext.http.protocol}://${event.requestContext.domainName}/${event.rawPath}${event.rawQueryString}`);

  /** @type {Request} */
  const req = new Request(url, {
    method: event.requestContext.http.method,
    headers: new Headers(event.headers),
    body: event.body,
  })

  /** @type {Response} */
  const res = await server.respond(req, {
    platform: { context },
    getClientAddress: (request) => {
      return event.headers["x-forwarded-for"] || event.requestContext.http.sourceIp;
    }
  })

  return {
    statusCode: res.status,
    headers: res.headers,
    // Object.fromEntries(res.headers.entries())
    body: await res.text(),
    isBase64Encoded: false,
  }
};