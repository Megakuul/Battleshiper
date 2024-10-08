import { AdapterError } from "../error";

/**
 * @typedef {Object} buildProjectInput
 * @property {string} project_name
 */

/**
 * @typedef {Object} buildProjectOutput
 * @property {string} message
 */

/**
 * Manually triggers a project build.
 * @param {buildProjectInput} input
 * @returns {Promise<buildProjectOutput>}
 * @throws {AdapterError}
 */
export const BuildProject = async (input) => {
  const res = await fetch("/api/resource/buildproject", {
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