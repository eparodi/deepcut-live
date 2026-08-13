import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ChatInput } from "./ChatInput";

describe("ChatInput", () => {
  const signInUrl = "http://localhost:8081/api/auth/google";

  it("renders sign-in prompt when not signed in", () => {
    render(
      <ChatInput
        isSignedIn={false}
        isReconnecting={false}
        isStreamEnded={false}
        isRateLimited={false}
        signInUrl={signInUrl}
        onSend={vi.fn()}
      />
    );
    expect(screen.getByText("Sign in to chat")).toBeInTheDocument();
  });

  it("renders 'Chat closed' when stream ended", () => {
    render(
      <ChatInput
        isSignedIn
        isReconnecting={false}
        isStreamEnded
        isRateLimited={false}
        signInUrl={signInUrl}
        onSend={vi.fn()}
      />
    );
    expect(
      screen.getByText("Chat closed — stream ended")
    ).toBeInTheDocument();
  });

  it("renders reconnecting message", () => {
    render(
      <ChatInput
        isSignedIn
        isReconnecting
        isStreamEnded={false}
        isRateLimited={false}
        signInUrl={signInUrl}
        onSend={vi.fn()}
      />
    );
    expect(screen.getByText("Reconnecting...")).toBeInTheDocument();
  });

  it("renders rate limit warning", () => {
    render(
      <ChatInput
        isSignedIn
        isReconnecting={false}
        isStreamEnded={false}
        isRateLimited
        rateLimitSeconds={3}
        signInUrl={signInUrl}
        onSend={vi.fn()}
      />
    );
    expect(
      screen.getByText(/Wait 3s before sending again/)
    ).toBeInTheDocument();
  });

  it("renders input and send button when signed in", () => {
    const onSend = vi.fn();
    render(
      <ChatInput
        isSignedIn
        isReconnecting={false}
        isStreamEnded={false}
        isRateLimited={false}
        signInUrl={signInUrl}
        onSend={onSend}
      />
    );
    expect(
      screen.getByPlaceholderText("Type a message...")
    ).toBeInTheDocument();
    expect(screen.getByText("Send")).toBeInTheDocument();
  });

  it("calls onSend when typing and pressing Enter", async () => {
    const user = userEvent.setup();
    const onSend = vi.fn();
    render(
      <ChatInput
        isSignedIn
        isReconnecting={false}
        isStreamEnded={false}
        isRateLimited={false}
        signInUrl={signInUrl}
        onSend={onSend}
      />
    );
    const input = screen.getByPlaceholderText("Type a message...");
    await user.type(input, "Hello chat{Enter}");
    expect(onSend).toHaveBeenCalledWith("Hello chat");
  });

  it("calls onSend when clicking Send button", async () => {
    const user = userEvent.setup();
    const onSend = vi.fn();
    render(
      <ChatInput
        isSignedIn
        isReconnecting={false}
        isStreamEnded={false}
        isRateLimited={false}
        signInUrl={signInUrl}
        onSend={onSend}
      />
    );
    const input = screen.getByPlaceholderText("Type a message...");
    await user.type(input, "Hello!");
    await user.click(screen.getByText("Send"));
    expect(onSend).toHaveBeenCalledWith("Hello!");
  });

  it("does not send empty messages", async () => {
    const user = userEvent.setup();
    const onSend = vi.fn();
    render(
      <ChatInput
        isSignedIn
        isReconnecting={false}
        isStreamEnded={false}
        isRateLimited={false}
        signInUrl={signInUrl}
        onSend={onSend}
      />
    );
    await user.click(screen.getByText("Send"));
    expect(onSend).not.toHaveBeenCalled();
  });

  it("clears the input when onSend reports success", async () => {
    const user = userEvent.setup();
    const onSend = vi.fn().mockReturnValue(true);
    render(
      <ChatInput
        isSignedIn
        isReconnecting={false}
        isStreamEnded={false}
        isRateLimited={false}
        signInUrl={signInUrl}
        onSend={onSend}
      />
    );
    const input = screen.getByPlaceholderText("Type a message...");
    await user.type(input, "Hello!");
    await user.click(screen.getByText("Send"));
    expect(onSend).toHaveBeenCalledWith("Hello!");
    expect(input).toHaveValue("");
  });

  it("keeps the text when onSend reports failure", async () => {
    const user = userEvent.setup();
    const onSend = vi.fn().mockReturnValue(false);
    render(
      <ChatInput
        isSignedIn
        isReconnecting={false}
        isStreamEnded={false}
        isRateLimited={false}
        signInUrl={signInUrl}
        onSend={onSend}
      />
    );
    const input = screen.getByPlaceholderText("Type a message...");
    await user.type(input, "Hello!");
    await user.click(screen.getByText("Send"));
    expect(onSend).toHaveBeenCalledWith("Hello!");
    expect(input).toHaveValue("Hello!");
  });

  it("disables input when reconnecting", () => {
    render(
      <ChatInput
        isSignedIn
        isReconnecting
        isStreamEnded={false}
        isRateLimited={false}
        signInUrl={signInUrl}
        onSend={vi.fn()}
      />
    );
    // Reconnecting shows a different UI, not the input
    expect(
      screen.queryByPlaceholderText("Type a message...")
    ).not.toBeInTheDocument();
  });
});
