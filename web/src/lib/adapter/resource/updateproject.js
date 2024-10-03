import { AdapterError } from "../error";

/**
 * @typedef {Object} repositoryInput
 * @property {number} id
 * @property {string} url
 * @property {string} branch
 */

/**
 * @typedef {Object} updateProjectInput
 * @property {string} project_name
 * @property {string} build_command
 * @property {string} output_directory
 * @property {repositoryInput} repository
 */

/**
 * @typedef {Object} updateProjectOutput
 * @property {string} message
 */

/**
 * Updates the project.
 * @param {updateProjectInput} input
 * @returns {Promise<updateProjectOutput>}
 * @throws {AdapterError}
 */
export const UpdateProject = async (input) => {
  const devUrl = import.meta.env.VITE_DEV_API_URL;
  const res = await fetch("/api/resource/updateproject", {
    method: "PATCH",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify(input),
  })
  if (res.ok) {
    return await res.json();
  } else {
    throw new AdapterError(await res.text(), res.status);
  }
}