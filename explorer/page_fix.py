path = '/Users/majedurrahman/Sovereign/explorer/app/page.tsx'
with open(path) as f:
    content = f.read()

old = """    } catch (err) {
      console.warn("Failed to fetch from real API. Falling back to simulated devnet data.", err);
      // Fallback mock data
      setBlocks([
        { height: 120532, time: new Date().toISOString(), proposer: "sovereignvaloper1x...", txCount: 3, gasUsed: 150000 },
        { height: 120531, time: new Date(Date.now() - 3000).toISOString(), proposer: "sovereignvaloper1y...", txCount: 0, gasUsed: 0 },
        { height: 120530, time: new Date(Date.now() - 6000).toISOString(), proposer: "sovereignvaloper1z...", txCount: 1, gasUsed: 50000 },
        { height: 120529, time: new Date(Date.now() - 9000).toISOString(), proposer: "sovereignvaloper1x...", txCount: 4, gasUsed: 210000 },
        { height: 120528, time: new Date(Date.now() - 12000).toISOString(), proposer: "sovereignvaloper1w...", txCount: 2, gasUsed: 100000 },
      ]);
      setTxs([
        { hash: "7c28f9d6ae1234c...", height: 120532, time: new Date().toISOString(), type: "cosmos", msgTypes: ["/cosmos.bank.v1beta1.MsgSend"], status: 0, fee: 150 },
        { hash: "8d92a10be43210b...", height: 120532, time: new Date().toISOString(), type: "cosmos", msgTypes: ["/cosmos.staking.v1beta1.MsgDelegate"], status: 0, fee: 250 },
        { hash: "0x3f5c9e2b1d7a8d...", height: 120530, time: new Date(Date.now() - 6000).toISOString(), type: "evm", msgTypes: ["EVMContractCall"], status: 0, fee: 500 },
        { hash: "9e8a7b6c5d4e3f2...", height: 120529, time: new Date(Date.now() - 9000).toISOString(), type: "cosmwasm", msgTypes: ["MsgExecuteContract"], status: 1, fee: 350 },
      ]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchDashboardData();"""

new = """    } catch (err) {
      console.warn("Failed to fetch dashboard data.", err);
      setError("Unable to reach explorer API. Please check the API service.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchDashboardData();
    const interval = setInterval(fetchDashboardData, 5000);
    return () => clearInterval(interval);
  }, []);"""

if old in content:
    content = content.replace(old, new)
    with open(path, 'w') as f:
        f.write(content)
    print('Replaced mock fallback successfully')
else:
    print('Pattern not found')
