import { splitProps, type JSX } from "solid-js";
import styles from "./Button.module.css";

type Variant = "primary" | "gray" | "red" | "green";

type ButtonProps = JSX.ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: Variant;
};

export default function Button(props: ButtonProps) {
  const [local, rest] = splitProps(props, ["variant", "class"]);
  const variant = () => local.variant ?? "primary";
  return (
    <button
      class={`${styles.btn} ${styles[variant()]}${local.class ? ` ${local.class}` : ""}`}
      {...rest}
    />
  );
}
