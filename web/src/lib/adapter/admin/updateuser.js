/**
 * @typedef {Object} updateInput
 * @property {string} subscription_id
 */

/**
 * @typedef {Object} updateUserInput
 * @property {string} user_id
 * @property {updateInput} update
 */

/**
 * @typedef {Object} updateUserOutput
 * @property {string} message
 */


/**
 * Updates a user.
 * @param {updateUserInput} input
 * @returns {Promise<updateUserOutput>}
 * @throws {Error}
 */
export const UpdateUser = async (input) => {
  const devUrl = import.meta.env.VITE_DEV_API_URL;
  const res = await fetch(`${devUrl?devUrl:""}/api/admin/updateuser`, {
    method: "PATCH",
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