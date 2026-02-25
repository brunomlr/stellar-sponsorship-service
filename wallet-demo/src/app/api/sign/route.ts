import { NextRequest, NextResponse } from "next/server";

export async function POST(request: NextRequest) {
  const apiUrl = process.env.SPONSORSHIP_API_URL;
  const apiKey = process.env.SPONSORSHIP_API_KEY;

  if (!apiUrl || !apiKey) {
    return NextResponse.json(
      { error: "config_error", message: "Server not configured" },
      { status: 500 }
    );
  }

  let body: unknown;
  try {
    body = await request.json();
  } catch {
    return NextResponse.json(
      { error: "invalid_request", message: "Invalid JSON body" },
      { status: 400 }
    );
  }

  try {
    const res = await fetch(`${apiUrl}/v1/sign`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${apiKey}`,
      },
      body: JSON.stringify(body),
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
