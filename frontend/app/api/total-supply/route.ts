import { NextResponse } from "next/server";

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const format = searchParams.get("format") || "raw";

  let totalSupply = "100000000"; // Fallback mainnet total supply: 100M WSOV

  try {
    const res = await fetch("http://localhost:8080/api/rest/cosmos/bank/v1beta1/supply/by_denom?denom=uwsov", {
      next: { revalidate: 60 }, // Cache for 60 seconds
    });
    if (res.ok) {
      const data = await res.json();
      if (data && data.amount && data.amount.amount) {
        // Convert from micro-units (6 decimals)
        const amountMicro = BigInt(data.amount.amount);
        const amountDec = Number(amountMicro) / 1000000;
        totalSupply = amountDec.toString();
      }
    }
  } catch (e) {
    console.error("Error fetching total supply from chain:", e);
  }

  if (format === "json") {
    return NextResponse.json({
      denom: "uwsov",
      amount: totalSupply,
      decimals: 6,
    });
  }

  return new Response(totalSupply, {
    headers: { "Content-Type": "text/plain" },
  });
}
