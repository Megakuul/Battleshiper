/**
 * @typedef {Object} pipelineSpecsInput
 * @property {number} daily_builds
 * @property {number} daily_deployments
 */

/**
 * @typedef {Object} projectSpecsInput
 * @property {number} project_count
 * @property {number} alias_count
 * @property {number} prerender_routes
 * @property {number} server_storage
 * @property {number} client_storage
 * @property {number} prerender_storage
 */

/**
 * @typedef {Object} cdnSpecsInput
 * @property {number} instance_count
 */

/**
 * @typedef {Object} upsertSubscriptionInput
 * @property {string} id
 * @property {string} name
 * @property {pipelineSpecsInput} pipeline_specs
 * @property {projectSpecsInput} project_specs
 * @property {cdnSpecsInput} cdn_specs
 */

/**
 * @typedef {Object} upsertSubscriptionOutput
 * @property {string} message
 */

/**
 * Upserts a subscription.
 * @param {upsertSubscriptionInput} input
 * @returns {Promise<upsertSubscriptionOutput>}
 * @throws {AdapterError}
 */
export const UpsertSubscription = async (input) => {
  const res = await fetch("/api/admin/upsertsubscription", {
    method: "PUT",
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