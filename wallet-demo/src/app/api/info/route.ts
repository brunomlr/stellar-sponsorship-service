import { NextResponse } from "next/server";

export async function GET() {
  const apiUrl = process.env.SPONSORSHIP_API_URL;
  if (!apiUrl) {
    return NextResponse.json(
      { error: "config_error", message: "SPONSORSHIP_API_URL not configured" },
      { status: 500 }
    );
  }

  try {
    const res = await fetch(`${apiUrl}/v1/info`, { cache: "no-store" });
    const data = await res.json();
    return NextResponse.json(data, { status: res.status });
  } catch (err) {
    return NextResponse.json(
      {
        error: "upstream_error",
        message:
          err instanceof Error ? err.message : "Failed to reach sponsorship API",
      },
      { status: 502 }
    );
  }
}
