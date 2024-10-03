import { AdapterError } from "../error";

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
 * @throws {AdapterError}
 */
export const UpdateUser = async (input) => {
  const res = await fetch("/api/admin/updateuser", {
    method: "PATCH",
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