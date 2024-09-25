/**
 * Redirects user to authorization route, which will redirect the user to the oauth provider.
 */
export const Authorize = () => {
  window.location.href = "/api/auth/authorize";
}