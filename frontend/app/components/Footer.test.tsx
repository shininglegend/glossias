import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { Footer } from "./Footer";

describe("Footer Component", () => {
  it("renders Glossias description and title", () => {
    render(<Footer />);
    expect(screen.getByText("Glossias")).toBeInTheDocument();
    expect(
      screen.getByText(
        /Interactive language learning through immersive stories/i,
      ),
    ).toBeInTheDocument();
  });

  it("renders key navigation and support links", () => {
    render(<Footer />);

    // Check story phase list
    expect(screen.getByText("Identify")).toBeInTheDocument();
    expect(screen.getByText("Recall")).toBeInTheDocument();
    expect(screen.queryByText("Grammar")).not.toBeInTheDocument();
    expect(screen.queryByText("Vocabulary")).not.toBeInTheDocument();

    // Check Support links
    const statusLink = screen.getByText("Status Page");
    expect(statusLink).toBeInTheDocument();
    expect(statusLink.getAttribute("href")).toBe(
      "https://status.glossias.org/",
    );

    const privacyLink = screen.getByText("Privacy Policy");
    expect(privacyLink).toBeInTheDocument();
    expect(privacyLink.getAttribute("href")).toBe("/privacy-policy");
  });

  it("renders the copyright text", () => {
    render(<Footer />);
    expect(
      screen.getByText(/Titus M\. All rights reserved\./i),
    ).toBeInTheDocument();
  });
});
