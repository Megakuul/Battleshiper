import 'SHIMS';
import { Server } from 'SERVER';
import { manifest, prerendered } from 'MANIFEST';

const server = new Server(manifest);

/**
 * @typedef {Object} AdapterRequest
 * @property {string} method
 * @property {string} path
 * @property {Object.<string, string>} headers
 * @property {string} body
 */

/**
 * @typedef {Object} AdapterResponse
 * @property {number} status_code
 * @property {string} status_description
 * @property {Object.<string, string>} headers
 * @property {string} body
 */

/**
 * @param {AdapterRequest} event
 * @param {import('aws-lambda').Context} context
 * @returns {AdapterResponse}
 */
export const handler = async (event, context) => {
  await server.init({ env: process.env });
  /** @type {Request} */
  const req = new Request(event.path, {
    method: event.method,
    headers: event.headers,
    body: event.body,
  })

  /** @type {Response} */
  const res = await server.respond(req, {
    platform: { context },
    getClientAddress: (request) => {
      return event.headers["x-forwarded-for"];
    }
  })

  return {
    statusCode: res.status,
    status_description: res.statusText,
    headers: res.headers,
    body: await res.text(),
  }
};