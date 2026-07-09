import { ethers } from "ethers";
import * as fs from "fs";
import * as path from "path";

const RPC_URL = "http://localhost:8545";
const MNEMONIC = "test test test test test test test test test test test junk";
const RECIPIENT = "0xe1f1a5093254b350c55514f8b9dbb40b996170c4"; // cosmos1u8c62zfj2je4p324znutnka5pwvkzuxyyk63dz

async function main() {
    console.log("Connecting to EVM RPC:", RPC_URL);
    const provider = new ethers.JsonRpcProvider(RPC_URL);

    // Derive wallet using standard BIP44 path for Cosmos-EVM
    const wallet = ethers.HDNodeWallet.fromPhrase(MNEMONIC);
    const signer = wallet.connect(provider);
    console.log("Faucet EVM Address:", signer.address);

    const buildDir = path.join(process.cwd(), "../evm/build");

    let nonce = await signer.getNonce();
    console.log("Current nonce:", nonce);

    // Helper to deploy contract
    const deployContract = async (name: string, args: any[] = []) => {
        console.log(`\nDeploying ${name}...`);
        const abi = JSON.parse(fs.readFileSync(path.join(buildDir, `${name}.abi`), "utf8"));
        const bytecode = fs.readFileSync(path.join(buildDir, `${name}.bin`), "utf8").trim();

        const factory = new ethers.ContractFactory(abi, "0x" + bytecode, signer);
        const contract = await factory.deploy(...args, {
            gasPrice: ethers.parseUnits("1", "gwei"),
            gasLimit: 3000000,
            nonce: nonce++
        });
        await contract.waitForDeployment();
        const address = await contract.getAddress();
        console.log(`${name} deployed at:`, address);
        return { contract, address };
    };

    // 1. Deploy ERC-20
    const { contract: erc20Raw, address: erc20Address } = await deployContract("TestERC20");
    const erc20 = erc20Raw as any;

    // 2. Deploy ERC-721
    const { contract: erc721Raw, address: erc721Address } = await deployContract("TestERC721");
    const erc721 = erc721Raw as any;

    // 3. Deploy ERC-1155
    const { contract: erc1155Raw, address: erc1155Address } = await deployContract("TestERC1155");
    const erc1155 = erc1155Raw as any;

    // 4. Deploy ERC-4626 (wrapping ERC-20)
    const { contract: erc4626Raw, address: erc4626Address } = await deployContract("TestERC4626", [erc20Address]);
    const erc4626 = erc4626Raw as any;

    // --- Mint and Transfer to recipient ---
    console.log("\nMinting and transferring tokens to recipient:", RECIPIENT);
    
    // ERC-20 transfer
    const tx20 = await erc20.transfer(RECIPIENT, ethers.parseEther("1500"), {
        gasPrice: ethers.parseUnits("1", "gwei"),
        gasLimit: 500000,
        nonce: nonce++
    });
    await tx20.wait();
    console.log("Transferred 1500 TERC20");

    // ERC-721 mint
    const tx721 = await erc721.mint(RECIPIENT, {
        gasPrice: ethers.parseUnits("1", "gwei"),
        gasLimit: 500000,
        nonce: nonce++
    });
    await tx721.wait();
    console.log("Minted ERC-721 NFT to recipient");

    // ERC-1155 mint
    const tx1155_1 = await erc1155.mint(RECIPIENT, 99, 25, {
        gasPrice: ethers.parseUnits("1", "gwei"),
        gasLimit: 500000,
        nonce: nonce++
    });
    await tx1155_1.wait();
    const tx1155_2 = await erc1155.mint(RECIPIENT, 1001, 1, {
        gasPrice: ethers.parseUnits("1", "gwei"),
        gasLimit: 500000,
        nonce: nonce++
    }); // Collectible/NFT item
    await tx1155_2.wait();
    console.log("Minted ERC-1155 tokens to recipient");

    // ERC-4626 deposit
    // First approve vault
    const approveTx = await erc20.approve(erc4626Address, ethers.parseEther("500"), {
        gasPrice: ethers.parseUnits("1", "gwei"),
        gasLimit: 500000,
        nonce: nonce++
    });
    await approveTx.wait();
    // Deposit into vault
    const depositTx = await erc4626.deposit(ethers.parseEther("500"), RECIPIENT, {
        gasPrice: ethers.parseUnits("1", "gwei"),
        gasLimit: 1000000,
        nonce: nonce++
    });
    await depositTx.wait();
    console.log("Deposited 500 TERC20 into ERC-4626 Vault on behalf of recipient");

    console.log("\nDeployment and transfers complete!");
    console.log("ERC20_ADDRESS=" + erc20Address);
    console.log("ERC721_ADDRESS=" + erc721Address);
    console.log("ERC1155_ADDRESS=" + erc1155Address);
    console.log("ERC4626_ADDRESS=" + erc4626Address);
}

main().catch((err) => {
    console.error(err);
    process.exit(1);
});
