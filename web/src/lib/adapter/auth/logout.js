/**
 * Logout the user from the application.
 * @throws {Error}
 */
export const Logout = async () => {
  const devUrl = import.meta.env.VITE_DEV_API_URL;
  const res = await fetch(`${devUrl?devUrl:""}/api/auth/logout`, {
    method: "POST",
  })
  if (res.ok) {
    return;
  } else {
    throw new Error(await res.text());
  }
}