/**
 * @typedef {Object} buildProjectInput
 * @property {string} project_name
 */

/**
 * @typedef {Object} buildProjectOutput
 * @property {string} message
 */

/**
 * Manually triggers a project build.
 * @param {buildProjectInput} input
 * @returns {Promise<buildProjectOutput>}
 * @throws {Error}
 */
export const BuildProject = async (input) => {
  const devUrl = import.meta.env.VITE_DEV_API_URL;
  const res = await fetch(`${devUrl?devUrl:""}/api/resource/buildproject`, {
    method: "POST",
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