// 入口: 挂载 <App /> 到 #root.
import { html, render } from "./lib/deps.js";
import { App } from "./components/App.js";

const root = document.getElementById("root");
if (root) render(html`<${App} />`, root);
