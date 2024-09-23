/**
 * Registers the user on the database.
 * @throws {Error}
 */
export const RegisterUser = async () => {
  const devUrl = import.meta.env.VITE_DEV_API_URL;
  const res = await fetch(`${devUrl?devUrl:""}/api/user/registeruser`, {
    method: "POST",
  })
  if (res.ok) {
    return;
  } else {
    throw new Error(await res.text());
  }
}