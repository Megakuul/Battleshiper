/**
 * @typedef {Object} repositoryOutput
 * @property {number} id
 * @property {string} name
 * @property {string} full_name
 */

/**
 * @typedef {Object} listRepositoryOutput
 * @property {string} message
 * @property {repositoryOutput[]} repositories
 */

/**
 * Fetches all repositories that battleshiper github app has access to.
 * @returns {Promise<listRepositoryOutput>}
 * @throws {AdapterError}
 */
export const ListRepository = async () => {
  const res = await fetch("/api/resource/listrepository", {
    method: "GET",
  })
  if (res.ok) {
    return await res.json();
  } else {
    throw new AdapterError(await res.text(), res.status);
  }
}