/**
 * @typedef {Object} updateAliasInput
 * @property {string} project_name
 * @property {Object.<string, null>} aliases
 */

/**
 * @typedef {Object} updateAliasOutput
 * @property {string} message
 */

/**
 * Updates the project aliases.
 * @param {updateAliasInput} input
 * @returns {Promise<updateAliasOutput>}
 * @throws {Error}
 */
export const UpdateAlias = async (input) => {
  const res = await fetch("/api/resource/updatealias", {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify(input),
  })
  if (res.ok) {
    return await res.json();
  } else {
    throw new Error(await res.text());
  }
}