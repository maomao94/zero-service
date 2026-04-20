// Preact + htm 绑定. 所有 UI 文件统一从这里导入, 方便一次改版本.
import { h, render, createContext, Fragment } from "preact";
import {
  useState, useEffect, useRef, useCallback, useContext,
  useMemo, useReducer, useLayoutEffect,
} from "preact/hooks";
import htm from "htm";

export const html = htm.bind(h);
export {
  h, render, createContext, Fragment,
  useState, useEffect, useRef, useCallback, useContext,
  useMemo, useReducer, useLayoutEffect,
};
