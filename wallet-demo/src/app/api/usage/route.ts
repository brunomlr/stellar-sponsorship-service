import { NextResponse } from "next/server";

export async function GET() {
  const apiUrl = process.env.SPONSORSHIP_API_URL;
  const apiKey = process.env.SPONSORSHIP_API_KEY;

  if (!apiUrl || !apiKey) {
    return NextResponse.json(
      { error: "config_error", message: "Server not configured" },
      { status: 500 }
    );
  }

  try {
    const res = await fetch(`${apiUrl}/v1/usage`, {
      headers: { Authorization: `Bearer ${apiKey}` },
      cache: "no-store",
    });
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
