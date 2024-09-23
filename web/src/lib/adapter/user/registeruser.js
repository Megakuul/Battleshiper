import { Authorize } from "../auth/authorize";

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
  } else if (res.status === 401) {
    Authorize()
  } else {
    throw new Error(await res.text());
  }
}