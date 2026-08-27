import React from "react";
import { describe, it, expect, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import Modal from "./Modal";
import ConfirmDialog from "./ConfirmDialog";

describe("Modal", () => {
  it("renders nothing when closed", () => {
    render(
      <Modal isOpen={false} onClose={() => {}} title="Hidden">
        body
      </Modal>,
    );
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("is a modal dialog labelled by its title and described by its description", () => {
    render(
      <Modal isOpen onClose={() => {}} title="My title" description="Details">
        body
      </Modal>,
    );
    const dialog = screen.getByRole("dialog");
    expect(dialog.tagName).toBe("DIALOG");
    expect(dialog).toHaveAttribute("open");
    expect(dialog).toHaveAttribute("aria-modal", "true");
    expect(dialog).toHaveAccessibleName("My title");
    expect(dialog).toHaveAccessibleDescription("Details");
  });

  it("omits aria-describedby when there is no description", () => {
    render(
      <Modal isOpen onClose={() => {}} title="No description">
        body
      </Modal>,
    );
    expect(screen.getByRole("dialog")).not.toHaveAttribute("aria-describedby");
  });

  it("calls onClose on Escape (native cancel event)", () => {
    const onClose = vi.fn();
    render(
      <Modal isOpen onClose={onClose} title="Esc">
        body
      </Modal>,
    );
    fireEvent(
      screen.getByRole("dialog"),
      new Event("cancel", { cancelable: true }),
    );
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("ignores Escape and backdrop clicks while closeDisabled", () => {
    const onClose = vi.fn();
    render(
      <Modal isOpen onClose={onClose} title="Busy" closeDisabled>
        body
      </Modal>,
    );
    const dialog = screen.getByRole("dialog");
    fireEvent(dialog, new Event("cancel", { cancelable: true }));
    fireEvent.click(dialog);
    expect(onClose).not.toHaveBeenCalled();
  });

  it("closes on backdrop click but not on clicks inside the panel", () => {
    const onClose = vi.fn();
    render(
      <Modal isOpen onClose={onClose} title="Backdrop">
        <button>inside</button>
      </Modal>,
    );
    fireEvent.click(screen.getByText("inside"));
    expect(onClose).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("dialog"));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("does not close on backdrop click when closeOnBackdropClick is false", () => {
    const onClose = vi.fn();
    render(
      <Modal
        isOpen
        onClose={onClose}
        title="Sticky"
        closeOnBackdropClick={false}
      >
        body
      </Modal>,
    );
    fireEvent.click(screen.getByRole("dialog"));
    expect(onClose).not.toHaveBeenCalled();
  });

  it("focuses initialFocusRef on open and restores focus on close", () => {
    function Harness() {
      const [open, setOpen] = React.useState(false);
      const focusRef = React.useRef<HTMLButtonElement>(null);
      return (
        <>
          <button onClick={() => setOpen(true)}>open</button>
          <Modal
            isOpen={open}
            onClose={() => setOpen(false)}
            title="Focus"
            initialFocusRef={focusRef}
          >
            <button>first</button>
            <button ref={focusRef} onClick={() => setOpen(false)}>
              target
            </button>
          </Modal>
        </>
      );
    }
    render(<Harness />);
    const trigger = screen.getByText("open");
    trigger.focus();
    expect(document.activeElement).toBe(trigger);

    fireEvent.click(trigger);
    expect(document.activeElement).toBe(screen.getByText("target"));

    fireEvent.click(screen.getByText("target"));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(document.activeElement).toBe(trigger);
  });
});

describe("ConfirmDialog", () => {
  it("exposes title and message to assistive tech and focuses the confirm button", () => {
    render(
      <ConfirmDialog
        isOpen
        onClose={() => {}}
        onConfirm={() => {}}
        variant="delete"
        title="Delete All Audio"
        message="Gone forever."
      />,
    );
    const dialog = screen.getByRole("dialog");
    expect(dialog).toHaveAccessibleName("Delete All Audio");
    expect(dialog).toHaveAccessibleDescription("Gone forever.");
    expect(document.activeElement).toBe(
      screen.getByRole("button", { name: "Delete" }),
    );
  });

  it("wires confirm/cancel and blocks Escape while loading", () => {
    const onClose = vi.fn();
    const onConfirm = vi.fn();
    const { rerender } = render(
      <ConfirmDialog isOpen onClose={onClose} onConfirm={onConfirm} />,
    );
    fireEvent.click(screen.getByRole("button", { name: "Confirm" }));
    expect(onConfirm).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onClose).toHaveBeenCalledTimes(1);

    rerender(
      <ConfirmDialog isOpen onClose={onClose} onConfirm={onConfirm} loading />,
    );
    fireEvent(
      screen.getByRole("dialog"),
      new Event("cancel", { cancelable: true }),
    );
    expect(onClose).toHaveBeenCalledTimes(1);
    expect(
      screen.getByRole("button", { name: "Processing..." }),
    ).toBeDisabled();
  });
});
