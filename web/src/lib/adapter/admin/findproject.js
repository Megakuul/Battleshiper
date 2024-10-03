import { AdapterError } from "../error";

/**
 * @typedef {Object} findProjectInput
 * @property {string} project_name
 * @property {string} owner_id
 */

/**
 * @typedef {Object} repositoryOutput
 * @property {number} id
 * @property {string} url
 * @property {string} branch
 */

/**
 * @typedef {Object} projectOutput
 * @property {string} name
 * @property {boolean} deleted
 * @property {boolean} initialized
 * @property {string} status
 * @property {Object.<string, Object>} aliases
 * @property {repositoryOutput} repository
 * @property {string} owner_id
 */

/**
 * @typedef {Object} findProjectOutput
 * @property {string} message
 * @property {projectOutput[]} projects
 */

/**
 * Find the specified project on the database.
 * @param {findProjectInput} input
 * @returns {Promise<findProjectOutput>}
 * @throws {AdapterError}
 */
export const FindProject = async (input) => {
  const res = await fetch(`/api/admin/findproject?${new URLSearchParams(input).toString()}`, {
    method: "GET",
  })
  if (res.ok) {
    return await res.json();
  } else {
    throw new AdapterError(await res.text(), res.status);
  }
}