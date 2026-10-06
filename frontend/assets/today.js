// Today page: planned meals eaten and left, kcal left, and extras. Every change goes
// to the API, which answers with the whole day; the page just renders that.
import { el, fmt, mealLabel, request } from "./common.js";

const $ = (id) => document.getElementById(id);
const main = $("today");

let today = null;
let ingredients = null; // loaded on first use of the extras search
const recipesByMeal = new Map();

// kcal from macros, as the server computes it (4/4/9 kcal per gram)
const kcalOf = (p, c, f) => 4 * p + 4 * c + 9 * f;

function showError(id, msg) {
  $(id).textContent = msg;
  $(id).hidden = !msg;
}

// ---------- render ----------

function render() {
  main.dataset.state = "ready";

  const date = new Date(`${today.date}T12:00:00`);
  $("date").textContent = date.toLocaleDateString("en-GB", { weekday: "long", day: "numeric", month: "long" });

  const over = today.leftKcal < 0;
  $("left").textContent = String(Math.abs(today.leftKcal));
  $("left-label").textContent = over ? "kcal over" : "kcal left";
  main.classList.toggle("over", over);
  $("of").replaceChildren(el("b", {}, String(today.eatenKcal)), " of ", el("b", {}, String(today.targetKcal)), " kcal eaten");

  renderDayline();
  renderExtras();
  renderMeals();
}

// one segment per planned meal (by its share of the day), filled once eaten, then extras
function renderDayline() {
  const segs = today.meals
    .filter((m) => m.pct > 0)
    .map((m) => el("i", { className: m.eaten ? "done" : "", style: `flex: ${m.pct}` }));
  const extraKcal = today.extras.reduce((sum, e) => sum + e.kcal, 0)
    + today.meals.filter((m) => !m.pct && m.eaten).reduce((sum, m) => sum + m.eaten.kcal, 0);
  if (extraKcal > 0 && today.targetKcal > 0) {
    segs.push(el("i", { className: "extra", style: `flex: ${(extraKcal / today.targetKcal) * 100}` }));
  }
  $("dayline").replaceChildren(...segs);
}

function renderExtras() {
  $("extras").replaceChildren(
    ...today.extras.map((e) =>
      el("div", { className: "extra-row" },
        el("span", {}, e.name, el("small", {}, e.ingredientId ? `${fmt(e.grams)} g` : "by hand")),
        el("b", {}, `${fmt(e.kcal)} kcal`),
        el("button", {
          type: "button", className: "remove", title: "Remove", ariaLabel: `Remove ${e.name}`,
          onclick: () => change("DELETE", `/today/extras/${e.id}`, null, "extra-error"),
        }, "×"),
      ),
    ),
  );
  $("extras-empty").hidden = today.extras.length > 0;
}

function renderMeals() {
  $("meal-list").replaceChildren(...today.meals.map(mealCard));
}

function mealCard(m) {
  const label = mealLabel(m.name);
  if (!m.eaten) {
    return el("article", { className: "mealcard todo" },
      el("header", {},
        el("div", {}, el("h3", {}, label), el("div", { className: "state" }, "To eat")),
        el("div", { className: "k" }, String(m.planKcal), el("small", {}, "kcal")),
      ),
      el("button", { type: "button", className: "choose", onclick: () => openChooser(m) }, "Choose recipe"),
    );
  }
  const e = m.eaten;
  return el("article", { className: "mealcard" },
    el("header", {},
      el("div", {}, el("h3", {}, label), el("div", { className: "state done" }, "Eaten")),
      el("div", { className: "k" }, fmt(e.kcal), el("small", {}, "kcal")),
    ),
    el("p", { className: "recipe" }, e.recipeName),
    el("div", { className: "chips" },
      ...e.items.filter((it) => it.grams > 0).map((it) => el("span", {}, `${it.nameEn} `, el("b", {}, `${fmt(it.grams)} g`))),
    ),
    el("button", {
      type: "button", className: "link undo",
      onclick: () => change("DELETE", `/today/meals/${m.name}`, null, "meal-error"),
    }, "Undo"),
  );
}

// send a change; the answer is the updated day
async function change(method, path, body, errorId) {
  showError(errorId, "");
  try {
    today = await request(method, path, body);
    render();
    return true;
  } catch (err) {
    showError(errorId, err.message);
    return false;
  }
}

// ---------- recipe chooser ----------

let chooser = null; // { meal, recipes, selected }

async function openChooser(m) {
  chooser = { meal: m, recipes: [], selected: null };
  $("chooser-title").textContent = `${mealLabel(m.name)} · ${m.planKcal} kcal`;
  $("recipe-search").value = "";
  $("preview").hidden = true;
  showError("chooser-error", "");
  $("meal-list").hidden = true;
  $("chooser").hidden = false;
  $("recipe-search").focus();

  try {
    if (!recipesByMeal.has(m.name)) recipesByMeal.set(m.name, await request("GET", `/recipes/meal/${m.name}`));
    chooser.recipes = recipesByMeal.get(m.name);
    renderRecipes();
  } catch (err) {
    showError("chooser-error", err.message);
  }
}

function closeChooser() {
  chooser = null;
  $("chooser").hidden = true;
  $("meal-list").hidden = false;
}

function renderRecipes() {
  const q = $("recipe-search").value.trim().toLowerCase();
  const list = chooser.recipes.filter((r) => r.name.toLowerCase().includes(q));
  $("recipes").replaceChildren(
    ...list.map((r) =>
      el("li", {}, el("button", {
        type: "button", role: "option",
        className: chooser.selected === r._id ? "sel" : "",
        ariaSelected: String(chooser.selected === r._id),
        onclick: () => pickRecipe(r._id),
      }, r.name.trim())),
    ),
  );
  if (!list.length) $("recipes").replaceChildren(el("li", { className: "hint" }, "No recipe matches."));
}

async function pickRecipe(id) {
  chooser.selected = id;
  renderRecipes();
  showError("chooser-error", "");
  try {
    const p = await request("GET", `/today/meals/${chooser.meal.name}/portion?recipe=${encodeURIComponent(id)}`);
    if (chooser?.selected !== id) return; // another recipe was picked meanwhile
    $("preview-kcal").textContent = `${fmt(p.kcal)} kcal`;
    $("preview-items").replaceChildren(
      ...p.items.filter((it) => it.grams > 0).flatMap((it) => [
        el("dt", { className: it.fixed ? "fx" : "" }, it.nameEn),
        el("dd", { className: it.fixed ? "fx" : "" }, `${fmt(it.grams)} g`),
      ]),
    );
    $("preview").hidden = false;
  } catch (err) {
    showError("chooser-error", err.message);
  }
}

$("recipe-search").addEventListener("input", () => chooser && renderRecipes());
$("chooser-cancel").addEventListener("click", closeChooser);
$("eat").addEventListener("click", async () => {
  if (!chooser?.selected) return;
  $("eat").disabled = true;
  const ok = await change("PUT", `/today/meals/${chooser.meal.name}`, { recipeId: chooser.selected }, "chooser-error");
  $("eat").disabled = false;
  if (ok) closeChooser();
});

// ---------- extras ----------

const search = $("extra-search");
const suggestions = $("suggestions");
let picked = null; // ingredient chosen for the amount step
let mode = "search"; // "search" | "amount" | "manual"
let active = -1; // highlighted suggestion

function setMode(next) {
  mode = next;
  $("extra-pick").hidden = next !== "search";
  $("extra-amount").hidden = next !== "amount";
  $("extra-manual").hidden = next !== "manual";
  showError("extra-error", "");
}

async function loadIngredients() {
  if (ingredients) return;
  try {
    ingredients = await request("GET", "/ingredients");
    for (const i of ingredients) {
      i.kcal100 = kcalOf(i.protein, i.carbs, i.fats);
      i.search = `${i.nameEn} ${i.name}`.toLowerCase();
    }
  } catch (err) {
    showError("extra-error", err.message);
  }
}

function suggestionOptions() {
  const q = search.value.trim().toLowerCase();
  if (!q) return [];
  const hits = (ingredients || []).filter((i) => i.search.includes(q)).slice(0, 6);
  return [...hits.map((i) => ({ ingredient: i })), { manual: search.value.trim() }];
}

function renderSuggestions() {
  const opts = suggestionOptions();
  active = Math.min(active, opts.length - 1);
  suggestions.replaceChildren(
    ...opts.map((o, i) =>
      el("li", {
        role: "option", id: `sugg-${i}`,
        className: `${o.manual !== undefined ? "manual" : ""} ${i === active ? "sel" : ""}`,
        ariaSelected: String(i === active),
        onmousedown: (e) => { e.preventDefault(); choose(o); },
      },
      o.manual !== undefined
        ? `Not in the list? Add "${o.manual}" by hand`
        : [o.ingredient.nameEn, el("em", {}, `${fmt(o.ingredient.kcal100)} kcal / 100 g`)]),
    ),
  );
  suggestions.hidden = opts.length === 0;
  search.setAttribute("aria-expanded", String(opts.length > 0));
  search.setAttribute("aria-activedescendant", active >= 0 ? `sugg-${active}` : "");
}

function choose(o) {
  suggestions.hidden = true;
  if (o.manual !== undefined) {
    setMode("manual");
    $("manual-name").value = o.manual;
    $("manual-kcal").value = "";
    $("manual-kcal").focus();
    return;
  }
  picked = o.ingredient;
  setMode("amount");
  $("picked-name").textContent = picked.nameEn;
  $("extra-grams").value = "";
  updateCalc();
  $("extra-grams").focus();
}

function updateCalc() {
  const g = Number($("extra-grams").value);
  if (!picked || !(g > 0)) {
    $("extra-calc").textContent = `${fmt(picked?.kcal100 ?? 0)} kcal per 100 g`;
    return;
  }
  const f = g / 100;
  $("extra-calc").replaceChildren("= ", el("b", {}, `${fmt(picked.kcal100 * f)} kcal`), ` · ${fmt(picked.protein * f)} g protein`);
}

function resetExtraForm() {
  picked = null;
  search.value = "";
  active = -1;
  renderSuggestions();
  setMode("search");
  search.focus(); // ready for the next one
}

search.addEventListener("focus", loadIngredients);
search.addEventListener("input", async () => {
  await loadIngredients();
  active = 0;
  renderSuggestions();
});
search.addEventListener("keydown", (e) => {
  const opts = suggestionOptions();
  if (!opts.length) return;
  if (e.key === "ArrowDown" || e.key === "ArrowUp") {
    e.preventDefault();
    active = (active + (e.key === "ArrowDown" ? 1 : -1) + opts.length) % opts.length;
    renderSuggestions();
  } else if (e.key === "Enter") {
    e.preventDefault();
    if (active >= 0) choose(opts[active]);
  } else if (e.key === "Escape") {
    suggestions.hidden = true;
  }
});
search.addEventListener("blur", () => { suggestions.hidden = true; });
$("extra-grams").addEventListener("input", updateCalc);
$("picked-change").addEventListener("click", () => { setMode("search"); search.focus(); renderSuggestions(); });
$("manual-cancel").addEventListener("click", () => { setMode("search"); search.focus(); });

$("extra-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  let body;
  if (mode === "amount") {
    const grams = Number($("extra-grams").value);
    if (!(grams > 0)) return showError("extra-error", "Enter how many grams you ate.");
    body = { ingredientId: picked._id, grams };
  } else if (mode === "manual") {
    const name = $("manual-name").value.trim();
    const kcal = Number($("manual-kcal").value);
    if (!name) return showError("extra-error", "Give it a name.");
    if (!(kcal > 0)) return showError("extra-error", "Enter its kcal.");
    body = { name, kcal };
  } else {
    return;
  }
  if (await change("POST", "/today/extras", body, "extra-error")) resetExtraForm();
});

// ---------- start ----------

async function load() {
  try {
    today = await request("GET", "/today");
  } catch {
    main.dataset.state = "ready";
    showError("meal-error", "Could not reach the server. Is the backend running?");
    return;
  }
  if (!today.targetKcal) {
    location.replace("/targets.html"); // first visit: set targets first
    return;
  }
  render();
}

load();
