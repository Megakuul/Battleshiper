/**
 * @typedef {Object} eventResultOutput
 * @property {string} execution_identifier
 * @property {number} timestamp
 * @property {boolean} successful
 */

/**
 * @typedef {Object} buildResultOutput
 * @property {string} execution_identifier
 * @property {number} timestamp
 * @property {boolean} successful
 */

/**
 * @typedef {Object} deploymentResultOutput
 * @property {string} execution_identifier
 * @property {number} timestamp
 * @property {boolean} successful
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
 * @property {string} build_image
 * @property {string} build_command
 * @property {string} output_directory
 * @property {repositoryOutput} repository
 * @property {Object.<string, null>} aliases
 * @property {eventResultOutput} last_event_result
 * @property {buildResultOutput} last_build_result
 * @property {deploymentResultOutput} last_deployment_result
 */

/**
 * @typedef {Object} listProjectOutput
 * @property {string} message
 * @property {projectOutput[]} projects
 */

/**
 * Fetches all projects.
 * @returns {Promise<listProjectOutput>}
 * @throws {AdapterError}
 */
export const ListProject = async () => {
  const res = await fetch("/api/resource/listproject", {
    method: "GET",
  })
  if (res.ok) {
    return await res.json();
  } else {
    throw new AdapterError(await res.text(), res.status);
  }
}