import { Authorize } from "../auth/authorize";

/**
 * Registers the user on the database.
 * @throws {AdapterError}
 */
export const RegisterUser = async () => {
  const res = await fetch("/api/user/registeruser", {
    method: "POST",
  })
  if (res.ok) {
    return;
  } else if (res.status === 401) {
    Authorize()
  } else {
    throw new AdapterError(await res.text(), res.status);
  }
}