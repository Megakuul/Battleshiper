/**
 * Refresh the user tokens (access_token & user_token).
 * @throws {Error}
 */
export const Refresh = async () => {
  const devUrl = import.meta.env.VITE_DEV_API_URL;
  const res = await fetch(`${devUrl?devUrl:""}/api/auth/refresh`, {
    method: "POST",
  })
  if (res.ok) {
    return;
  } else {
    throw new Error(await res.text());
  }
}