/**
 * @typedef {Object} repositoryInput
 * @property {number} id
 * @property {string} url
 * @property {string} branch
 */

/**
 * @typedef {Object} createProjectInput
 * @property {string} project_name
 * @property {string} build_image
 * @property {string} build_command
 * @property {string} output_directory
 * @property {repositoryInput} repository
 */

/**
 * @typedef {Object} createProjectOutput
 * @property {string} message
 */

/**
 * Creates a project.
 * @param {createProjectInput} input
 * @returns {Promise<createProjectOutput>}
 * @throws {AdapterError}
 */
export const CreateProject = async (input) => {
  const res = await fetch("/api/resource/createproject", {
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