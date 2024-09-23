/**
 * @typedef {Object} pipelineSpecsOutput
 * @property {number} daily_builds
 * @property {number} daily_deployments
 */

/**
 * @typedef {Object} projectSpecsOutput
 * @property {number} project_count
 * @property {number} alias_count
 * @property {number} prerender_routes
 * @property {number} server_storage
 * @property {number} client_storage
 * @property {number} prerender_storage
 */

/**
 * @typedef {Object} cdnSpecsOutput
 * @property {number} instance_count
 */

/**
 * @typedef {Object} subscriptionOutput
 * @property {string} id
 * @property {string} name
 * @property {pipelineSpecsOutput} pipeline_specs
 * @property {projectSpecsOutput} project_specs
 * @property {cdnSpecsOutput} cdn_specs
 */

/**
 * @typedef {Object} fetchInfoOutput
 * @property {string} id
 * @property {string} name
 * @property {Object.<ROLE, null>} roles
 * @property {string} provider
 * @property {string} avatar_url
 * @property {subscriptionOutput} [subscriptions]
 */

/**
 * Fetches the user profile informations.
 * @returns {Promise<fetchInfoOutput>}
 * @throws {Error}
 */
export const FetchInfo = async () => {
  const devUrl = import.meta.env.VITE_DEV_API_URL;
  const res = await fetch(`${devUrl?devUrl:""}/api/user/fetchinfo`, {
    method: "GET",
  })
  if (res.ok) {
    return await res.json();
  } else {
    throw new Error(await res.text());
  }
}