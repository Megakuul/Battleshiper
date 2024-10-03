import { AdapterError } from "../error";

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
 * @typedef {Object} listSubscriptionOutput
 * @property {string} message
 * @property {subscriptionOutput[]} subscriptions
 */

/**
 * Lists all subscriptions that exist.
 * @returns {Promise<listSubscriptionOutput>}
 * @throws {AdapterError}
 */
export const ListSubscription = async () => {
  const res = await fetch("/api/admin/listsubscription", {
    method: "GET",
  })
  if (res.ok) {
    return await res.json();
  } else {
    throw new AdapterError(await res.text(), res.status);
  }
}