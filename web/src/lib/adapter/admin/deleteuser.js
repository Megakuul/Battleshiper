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
export const DeleteUser = async (input) => {
  const res = await fetch("/api/admin/deleteuser", {
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