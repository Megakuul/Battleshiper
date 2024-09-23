/**
 * Redirects user to authorization route, which will redirect the user to the oauth provider.
 */
export const Authorize = () => {
  window.location.href = `${import.meta.env.VITE_DEV_API_URL}/api/auth/authorize`;
}