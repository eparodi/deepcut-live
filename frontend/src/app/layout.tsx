import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";

const inter = Inter({
  subsets: ["latin"],
  variable: "--font-inter",
});

export const metadata: Metadata = {
  title: "DeepCut Live — Stream What You Believe",
  description:
    "No censorship. No filters. A live streaming platform for free expression.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className={`${inter.variable} h-full antialiased`}>
      <body className="min-h-full bg-[var(--color-surface)] text-[var(--color-text)] flex flex-col">
        {children}
      </body>
    </html>
  );
}
