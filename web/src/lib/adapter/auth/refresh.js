/**
 * Refresh the user tokens (access_token & user_token).
 * @throws {AdapterError}
 */
export const Refresh = async () => {
  const res = await fetch("/api/auth/refresh", {
    method: "POST",
  })
  if (res.ok) {
    return;
  } else {
    throw new AdapterError(await res.text(), res.status);
  }
}