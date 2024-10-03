import { AdapterError } from "../error";

/**
 * Logout the user from the application.
 * @throws {AdapterError}
 */
export const Logout = async () => {
  const res = await fetch("/api/auth/logout", {
    method: "POST",
  })
  if (res.ok) {
    return;
  } else {
    throw new AdapterError(await res.text(), res.status);
  }
}