/**
 * @typedef {Record<string, string>} findUserInput
 * @property {string} user_id
 */

/**
 * @typedef {Object} userOutput
 * @property {string} id
 * @property {boolean} privileged
 * @property {string} provider
 * @property {Object.<string, Object>} roles
 * @property {string} subscription_id
 */

/**
 * @typedef {Object} findUserOutput
 * @property {string} message
 * @property {userOutput} user
 */


/**
 * Find the specified user on the database.
 * @param {findUserInput} input
 * @returns {Promise<findUserOutput>}
 * @throws {Error}
 */
export const FindUser = async (input) => {
  const devUrl = import.meta.env.VITE_DEV_API_URL;
  const res = await fetch(`${devUrl?devUrl:""}/api/admin/finduser?${new URLSearchParams(input).toString()}`, {
    method: "GET",
  })
  if (res.ok) {
    return await res.json();
  } else {
    throw new Error(await res.text());
  }
}