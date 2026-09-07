import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import type { ReactElement } from "react";
import {
  LegalPage,
  LEGAL_EFFECTIVE_DATE,
  LEGAL_CONTACT_EMAIL,
} from "./LegalPage";
import PrivacyPolicy from "~/routes/privacy-policy";
import TermsOfService from "~/routes/terms-of-service";

function renderInRouter(ui: ReactElement) {
  return render(<MemoryRouter>{ui}</MemoryRouter>);
}

describe("LegalPage", () => {
  it("renders the title, effective date, and footer", () => {
    renderInRouter(
      <LegalPage title="Example Policy">
        <p>Body copy</p>
      </LegalPage>,
    );

    expect(
      screen.getByRole("heading", { name: "Example Policy" }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(`Effective as of ${LEGAL_EFFECTIVE_DATE}`),
    ).toBeInTheDocument();
    expect(screen.getByText("Body copy")).toBeInTheDocument();
    expect(screen.getByText("Privacy Policy")).toBeInTheDocument();
  });
});

describe("Privacy Policy page", () => {
  it("names Glossias practices and the privacy contact", () => {
    renderInRouter(<PrivacyPolicy />);

    expect(
      screen.getByRole("heading", { name: "Privacy Policy" }),
    ).toBeInTheDocument();
    expect(screen.getAllByText(/Clerk/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/Anthropic/).length).toBeGreaterThan(0);
    expect(
      screen.getAllByRole("link", { name: LEGAL_CONTACT_EMAIL }).length,
    ).toBeGreaterThan(0);
    expect(
      screen.getAllByRole("link", { name: LEGAL_CONTACT_EMAIL })[0],
    ).toHaveAttribute("href", `mailto:${LEGAL_CONTACT_EMAIL}`);
  });
});

describe("Terms of Service page", () => {
  it("includes the arbitration opt-out and privacy link", () => {
    renderInRouter(<TermsOfService />);

    expect(
      screen.getByRole("heading", { name: "Terms of Service" }),
    ).toBeInTheDocument();
    expect(
      screen.getAllByText(/binding individual arbitration/i).length,
    ).toBeGreaterThan(0);
    expect(
      screen.getAllByRole("link", { name: "Privacy Policy" })[0],
    ).toHaveAttribute("href", "/privacy-policy");
  });
});
