/**
 * @typedef {Object} deleteProjectInput
 * @property {string} project_name
 */

/**
 * @typedef {Object} deleteProjectOutput
 * @property {string} message
 */

/**
 * Deletes a project.
 * @param {deleteProjectInput} input
 * @returns {Promise<deleteProjectOutput>}
 * @throws {Error}
 */
export const DeleteProject = async (input) => {
  const devUrl = import.meta.env.VITE_DEV_API_URL;
  const res = await fetch(`${devUrl?devUrl:""}/api/resource/deleteproject`, {
    method: "DELETE",
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