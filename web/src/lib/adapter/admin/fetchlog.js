import { AdapterError } from "../error";

/**
 * @typedef {Object} fetchLogInput
 * @property {string} log_type
 * @property {number} start_time
 * @property {number} end_time
 * @property {number} count
 * @property {boolean} filter_lambda
 */

/**
 * @typedef {Object} eventOutput
 * @property {number} timestamp
 * @property {string} message
 */

/**
 * @typedef {Object} fetchLogOutput
 * @property {string} message
 * @property {eventOutput[]} events
 */

/**
 * Fetches the logs of the battleshiper system.
 * @param {fetchLogInput} input
 * @returns {Promise<fetchLogOutput>}
 * @throws {AdapterError}
 */
export const FetchLog = async (input) => {
  const res = await fetch("/api/admin/fetchlog", {
    method: "POST",
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(input),
  })
  if (res.ok) {
    return await res.json();
  } else {
    throw new AdapterError(await res.text(), res.status);
  }
}