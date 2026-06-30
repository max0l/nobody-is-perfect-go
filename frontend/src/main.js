import App from "./App.svelte";
import "./app.css";
import { mount } from "svelte";

const app = mount(App, {
  target: document.getElementById("app"),
  props: {
    initialGameId: document.body.dataset.gameId || "",
  },
});

export default app;
