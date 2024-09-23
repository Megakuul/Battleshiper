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
 * @throws {Error}
 */
export const CreateProject = async (input) => {
  const devUrl = import.meta.env.VITE_DEV_API_URL;
  const res = await fetch(`${devUrl?devUrl:""}/api/resource/createproject`, {
    method: "POST",
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(input),
  })
  if (res.ok) {
    return await res.json();
  } else {
    throw new Error(await res.text());
  }
}