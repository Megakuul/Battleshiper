/**
 * @typedef {Object} deleteUserInput
 * @property {string} user_id
 */

/**
 * @typedef {Object} deleteUserOutput
 * @property {string} message
 */

/**
 * Deletes a user from the database.
 * @param {deleteUserInput} input
 * @returns {Promise<deleteUserOutput>}
 * @throws {Error}
 */
export const DeleteProject = async (input) => {
  const devUrl = import.meta.env.VITE_DEV_API_URL;
  const res = await fetch(`${devUrl?devUrl:""}/api/admin/deleteuser`, {
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