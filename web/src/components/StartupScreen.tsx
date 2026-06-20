import { motion } from "motion/react";

type StartupScreenProps = {
  shellTone: string;
  title?: string;
  subtitle?: string;
};

export function StartupScreen({
  shellTone,
  title = "Preparing workspace",
  subtitle = "Loading desktop environment",
}: StartupScreenProps) {
  return (
    <motion.div
      className={`login-shell ${shellTone}`}
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: 0.26, ease: "easeOut" }}
    >
      <motion.section
        className="login-card startup-card"
        initial={{ opacity: 0, y: 12, scale: 0.99 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        exit={{ opacity: 0, y: -10, scale: 0.985, filter: "blur(4px)" }}
        transition={{ duration: 0.34, ease: "easeOut" }}
      >
        <motion.div
          className="login-branding"
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.35, ease: "easeOut" }}
        >
          <img className="login-logo-image" src="/branding/veltryx_icon.png" alt="Veltryx icon" />
        </motion.div>
        <motion.p
          className="eyebrow"
          initial={{ opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.35, delay: 0.05, ease: "easeOut" }}
        >
          Veltryx
        </motion.p>
        <div className="startup-animation" aria-hidden="true">
          <motion.div
            className="startup-ring startup-ring-a"
            animate={{ rotate: 360 }}
            transition={{ duration: 1.35, ease: "linear", repeat: Infinity }}
          />
          <motion.div
            className="startup-ring startup-ring-b"
            animate={{ rotate: -360 }}
            transition={{ duration: 1.05, ease: "linear", repeat: Infinity }}
          />
          <motion.div
            className="startup-core"
            animate={{ scale: [0.98, 1.03, 0.98] }}
            transition={{ duration: 1.8, ease: "easeInOut", repeat: Infinity }}
          >
            <motion.div
              className="startup-pulse"
              animate={{ opacity: [0.18, 0.55, 0], scale: [0.78, 1.02, 1.28] }}
              transition={{ duration: 1.8, ease: "easeOut", repeat: Infinity }}
            />
          </motion.div>
        </div>
        <motion.h1
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.35, delay: 0.08, ease: "easeOut" }}
        >
          {title}
        </motion.h1>
        <motion.p
          className="login-text"
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.35, delay: 0.12, ease: "easeOut" }}
        >
          {subtitle}
        </motion.p>
      </motion.section>
    </motion.div>
  );
}
