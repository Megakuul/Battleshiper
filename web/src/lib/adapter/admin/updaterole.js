import { AdapterError } from "../error";

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
 * @throws {AdapterError}
 */
export const UpdateRole = async (input) => {
  const res = await fetch("/api/admin/updaterole", {
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