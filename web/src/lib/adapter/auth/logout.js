/**
 * Logout the user from the application.
 * @throws {Error}
 */
export const Logout = async () => {
  const res = await fetch("/api/auth/logout", {
    method: "POST",
  })
  if (res.ok) {
    return;
  } else {
    throw new Error(await res.text());
  }
}