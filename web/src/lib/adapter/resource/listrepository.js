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
 * @throws {Error}
 */
export const ListRepository = async () => {
  const devUrl = import.meta.env.VITE_DEV_API_URL;
  const res = await fetch(`${devUrl?devUrl:""}/api/resource/listrepository`, {
    method: "GET",
  })
  if (res.ok) {
    return await res.json();
  } else {
    throw new Error(await res.text());
  }
}