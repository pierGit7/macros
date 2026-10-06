// Helpers shared by the Today and Targets pages.

// request calls the backend through nginx's /api proxy (see nginx.conf) and returns
// the decoded JSON; a non-2xx answer throws with the server's error message.
export async function request(method, path, body) {
  const res = await fetch(`/api${path}`, {
    method,
    headers: body ? { "Content-Type": "application/json" } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || `Request failed (${res.status})`);
  return data;
}

export const fmt = (n) => Math.round(n).toLocaleString("en-US");

// "morningSnack" -> "Morning snack"
export function mealLabel(name) {
  const words = name.replace(/([A-Z])/g, " $1").toLowerCase();
  return words.charAt(0).toUpperCase() + words.slice(1);
}

// el builds an element: el("p", { className: "x" }, "text", child, [more, children])
export function el(tag, props = {}, ...children) {
  const node = Object.assign(document.createElement(tag), props);
  node.append(...children.flat().filter((c) => c != null && c !== false));
  return node;
}
