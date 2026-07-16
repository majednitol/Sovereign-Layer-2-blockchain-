import { NextResponse } from "next/server";

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const format = searchParams.get("format") || "raw";

  let totalSupply = 100000000; // 100M CSOV
  let treasuryBalance = 80000000; // 80M locked in treasury
  let reserveBalance = 5000000; // 5M locked in reserve fund

  try {
    // 1. Fetch total supply
    const supplyRes = await fetch("http://localhost:8080/api/rest/cosmos/bank/v1beta1/supply/by_denom?denom=uwsov", {
      next: { revalidate: 60 },
    });
    if (supplyRes.ok) {
      const data = await supplyRes.json();
      if (data && data.amount && data.amount.amount) {
        totalSupply = Number(BigInt(data.amount.amount)) / 1000000;
      }
    }
  } catch (e) {
    console.error("Error fetching total supply:", e);
  }

  try {
    // 2. Fetch treasury balance (cosmos1w8kmv94zcf8yysgw9dp8yzq6ffe2e8m0uj8dm0)
    const treasuryRes = await fetch("http://localhost:8080/api/rest/cosmos/bank/v1beta1/balances/cosmos1w8kmv94zcf8yysgw9dp8yzq6ffe2e8m0uj8dm0/by_denom?denom=uwsov", {
      next: { revalidate: 60 },
    });
    if (treasuryRes.ok) {
      const data = await treasuryRes.json();
      if (data && data.balance && data.balance.amount) {
        treasuryBalance = Number(BigInt(data.balance.amount)) / 1000000;
      }
    }
  } catch (e) {
    console.error("Error fetching treasury balance:", e);
  }

  try {
    // 3. Fetch reserve fund balance (cosmos1dag3w9ydhzmwpvd6asrt8elexa8s27ph7895jc)
    const reserveRes = await fetch("http://localhost:8080/api/rest/cosmos/bank/v1beta1/balances/cosmos1dag3w9ydhzmwpvd6asrt8elexa8s27ph7895jc/by_denom?denom=uwsov", {
      next: { revalidate: 60 },
    });
    if (reserveRes.ok) {
      const data = await reserveRes.json();
      if (data && data.balance && data.balance.amount) {
        reserveBalance = Number(BigInt(data.balance.amount)) / 1000000;
      }
    }
  } catch (e) {
    console.error("Error fetching reserve fund balance:", e);
  }

  const circulatingSupply = Math.max(0, totalSupply - treasuryBalance - reserveBalance).toString();

  if (format === "json") {
    return NextResponse.json({
      denom: "uwsov",
      amount: circulatingSupply,
      decimals: 6,
      totalSupply: totalSupply.toString(),
      lockedTreasury: treasuryBalance.toString(),
      lockedReserve: reserveBalance.toString(),
    });
  }

  return new Response(circulatingSupply, {
    headers: { "Content-Type": "text/plain" },
  });
}
