import { AnimatePresence, motion } from "motion/react";
import type { PageId } from "../types/app";
import { formatRoleLabel } from "../utils/app-helpers";

type SidebarItem = {
  id: PageId;
  label: string;
};

type ShellSidebarProps = {
  name: string;
  username: string;
  role: string;
  navGroups: Record<string, SidebarItem[]>;
  expandedSections: Record<string, boolean>;
  currentPage: PageId;
  onToggleSection: (section: string) => void;
  onSelectPage: (page: PageId) => void;
};

export function ShellSidebar({
  name,
  username,
  role,
  navGroups,
  expandedSections,
  currentPage,
  onToggleSection,
  onSelectPage,
}: ShellSidebarProps) {
  return (
    <aside className="workspace-sidebar">
      <div className="sidebar-user">
        <strong>{name}</strong>
        <p>
          @{username} | {formatRoleLabel(role)}
        </p>
      </div>

      {Object.entries(navGroups).map(([section, items]) => (
        <section className="nav-section" key={section}>
          <button
            className={`nav-section-toggle ${expandedSections[section] ? "open" : ""}`}
            onClick={() => onToggleSection(section)}
            type="button"
          >
            <span className="nav-section-title">{section}</span>
            <motion.span
              className="nav-section-icon"
              animate={{ rotate: expandedSections[section] ? 180 : 0 }}
              transition={{ duration: 0.2, ease: "easeOut" }}
            >
              +
            </motion.span>
          </button>
          <AnimatePresence initial={false}>
            {expandedSections[section] ? (
              <motion.div
                className="nav-list-wrap"
                initial={{ height: 0, opacity: 0, y: -6 }}
                animate={{ height: "auto", opacity: 1, y: 0 }}
                exit={{ height: 0, opacity: 0, y: -6 }}
                transition={{ duration: 0.22, ease: "easeOut" }}
              >
                <motion.div
                  className="nav-list"
                  initial="closed"
                  animate="open"
                  exit="closed"
                  variants={{
                    open: {
                      transition: { staggerChildren: 0.03, delayChildren: 0.02 },
                    },
                    closed: {
                      transition: { staggerChildren: 0.02, staggerDirection: -1 },
                    },
                  }}
                >
                  {items.map((item) => (
                    <motion.button
                      key={item.id}
                      className={`nav-link ${currentPage === item.id ? "active" : ""}`}
                      onClick={() => onSelectPage(item.id)}
                      variants={{
                        open: { opacity: 1, y: 0 },
                        closed: { opacity: 0, y: -4 },
                      }}
                      transition={{ duration: 0.16, ease: "easeOut" }}
                    >
                      {item.label}
                    </motion.button>
                  ))}
                </motion.div>
              </motion.div>
            ) : null}
          </AnimatePresence>
        </section>
      ))}
    </aside>
  );
}
