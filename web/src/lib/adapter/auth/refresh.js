/**
 * Refresh the user tokens (access_token & user_token).
 * @throws {Error}
 */
export const Refresh = async () => {
  const res = await fetch("/api/auth/refresh", {
    method: "POST",
  })
  if (res.ok) {
    return;
  } else {
    throw new Error(await res.text());
  }
}