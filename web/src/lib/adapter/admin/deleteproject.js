/**
 * @typedef {Object} deleteProjectInput
 * @property {string} project_name
 */

/**
 * @typedef {Object} deleteProjectOutput
 * @property {string} message
 */

/**
 * Deletes a project from the database.
 * @param {deleteProjectInput} input
 * @returns {Promise<deleteProjectOutput>}
 * @throws {AdapterError}
 */
export const DeleteProject = async (input) => {
  const res = await fetch("/api/admin/deleteproject", {
    method: "DELETE",
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