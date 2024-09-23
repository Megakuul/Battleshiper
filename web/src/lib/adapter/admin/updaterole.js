/**
 * @typedef {Object} updateRoleInput
 * @property {string} user_id
 * @property {Object.<ROLE, null>} rbac_roles
 */

/**
 * @typedef {Object} updateRoleOutput
 * @property {string} message
 */

/**
 * Updates the roles of a user.
 * @param {updateRoleInput} input
 * @returns {Promise<updateRoleOutput>}
 * @throws {Error}
 */
export const UpdateRole = async (input) => {
  const devUrl = import.meta.env.VITE_DEV_API_URL;
  const res = await fetch(`${devUrl?devUrl:""}/api/admin/updaterole`, {
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