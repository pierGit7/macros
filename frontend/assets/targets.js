// Targets page: load the daily target from the API, let the user change it, and
// render the kcal split. On the very first save it moves on to the Today page.
import { fmt, mealLabel, request } from "./common.js";

const form = document.getElementById("targets");
const panel = document.getElementById("panel");
const errorBox = document.getElementById("error");
const saveButton = form.querySelector("button[type=submit]");
const segButtons = [...form.querySelectorAll(".seg button")];
const fields = ["weight", "proteins", "carbs", "fats"];

let mealsPerDay = 4;
let firstRun = false; // no targets saved yet: after saving, go to Today
let edited = false; // set on first user input, so a slow initial GET never overwrites it
form.addEventListener("input", () => { edited = true; });

function setMealsPerDay(n) {
  mealsPerDay = n;
  for (const b of segButtons) b.setAttribute("aria-pressed", String(Number(b.dataset.meals) === n));
}

function fillForm(m) {
  for (const f of fields) form.elements[f].value = m[f] ? m[f] : "";
  setMealsPerDay(m.mealsPerDay);
}

function render(m) {
  if (!m.kcal) {
    panel.dataset.state = "empty";
    return;
  }
  panel.dataset.state = "ready";

  const kcal = { proteins: m.proteins * 4, carbs: m.carbs * 4, fats: m.fats * 9 };
  document.getElementById("kcal").textContent = String(Math.round(m.kcal)); // no separator: a comma looks off at this size in mono
  for (const f of ["proteins", "carbs", "fats"]) {
    document.getElementById(`bar-${f}`).style.flex = String(kcal[f] || 0.0001);
    document.getElementById(`g-${f}`).textContent = `${fmt(m[f])} g`;
    document.getElementById(`k-${f}`).textContent = `${fmt(kcal[f])} kcal`;
  }

  const meals = document.getElementById("meals");
  meals.style.setProperty("--meal-count", m.meals.length);
  meals.replaceChildren(
    ...m.meals.map((meal) => {
      const el = document.createElement("div");
      el.className = "meal";
      el.innerHTML = "<span></span><b></b><em></em>";
      el.children[0].textContent = mealLabel(meal.name);
      el.children[1].textContent = fmt(meal.kcal);
      el.children[2].textContent = `${meal.pct}%`;
      return el;
    }),
  );

  const ratio = document.getElementById("ratio");
  ratio.hidden = !(m.weight > 0 && m.proteins > 0);
  document.getElementById("ratio-value").textContent = `${(m.proteins / m.weight).toFixed(1)} g/kg`;
}

function showError(msg) {
  errorBox.textContent = msg;
  errorBox.hidden = !msg;
}

async function load() {
  try {
    const m = await request("GET", "/macros");
    firstRun = !m.kcal;
    if (!edited) fillForm(m);
    render(m);
  } catch {
    panel.dataset.state = "empty";
    showError("Could not reach the server. Is the backend running?");
  }
}

segButtons.forEach((b) =>
  b.addEventListener("click", () => {
    edited = true;
    setMealsPerDay(Number(b.dataset.meals));
  }),
);

form.addEventListener("submit", async (e) => {
  e.preventDefault();
  showError("");

  const body = { mealsPerDay };
  let valid = true;
  for (const f of fields) {
    const input = form.elements[f];
    const value = input.value === "" ? 0 : Number(input.value);
    const bad = !Number.isFinite(value) || value < 0;
    input.setAttribute("aria-invalid", String(bad));
    valid &&= !bad;
    body[f] = value;
  }
  if (!valid) {
    showError("Values must be zero or positive numbers.");
    return;
  }

  saveButton.disabled = true;
  try {
    const m = await request("PUT", "/macros", body);
    if (firstRun && m.kcal) {
      location.href = "/";
      return;
    }
    render(m);
    panel.classList.remove("updated");
    void panel.offsetWidth; // restart the fade-in animation
    panel.classList.add("updated");
  } catch (err) {
    showError(err.message);
  } finally {
    saveButton.disabled = false;
  }
});

load();
