//! Benchmark tests for VaultForge ZK policy verification.
//!
//! Run with: cargo bench --package zk

use criterion::{black_box, criterion_group, criterion_main, Criterion};
use vaultforge_zk::{PrivateInputs, PublicInputs, Prover, Verifier};

fn bench_zk_prove(c: &mut Criterion) {
    let private = PrivateInputs {
        daily_limit: 100_000,
        per_wallet_limit: 50_000,
        blinding_factor: [0x42u8; 32],
    };
    let public = PublicInputs {
        amount: 25_000,
        policy_version: "v1".to_string(),
        intent_id: "bench-intent-1".to_string(),
        wallet_id: "bench-wallet-1".to_string(),
    };

    c.bench_function("zk_prove_small_amount", |b| {
        b.iter(|| black_box(Prover::prove(black_box(&private), black_box(&public))))
    });

    let public_large = PublicInputs {
        amount: 49_999,
        policy_version: "v1".to_string(),
        intent_id: "bench-intent-2".to_string(),
        wallet_id: "bench-wallet-1".to_string(),
    };

    c.bench_function("zk_prove_near_limit", |b| {
        b.iter(|| black_box(Prover::prove(black_box(&private), black_box(&public_large))))
    });
}

fn bench_zk_verify(c: &mut Criterion) {
    let private = PrivateInputs {
        daily_limit: 100_000,
        per_wallet_limit: 50_000,
        blinding_factor: [0x42u8; 32],
    };
    let public = PublicInputs {
        amount: 25_000,
        policy_version: "v1".to_string(),
        intent_id: "bench-intent-1".to_string(),
        wallet_id: "bench-wallet-1".to_string(),
    };

    let proof = Prover::prove(&private, &public).unwrap();

    c.bench_function("zk_verify_proof", |b| {
        b.iter(|| black_box(Verifier::verify(black_box(&proof))))
    });
}

fn bench_zk_full_roundtrip(c: &mut Criterion) {
    c.bench_function("zk_full_prove_verify", |b| {
        b.iter(|| {
            let private = PrivateInputs {
                daily_limit: 100_000,
                per_wallet_limit: 50_000,
                blinding_factor: [0x99u8; 32],
            };
            let public = PublicInputs {
                amount: 25_000,
                policy_version: "v1".to_string(),
                intent_id: "bench-intent-rt".to_string(),
                wallet_id: "bench-wallet-rt".to_string(),
            };
            let proof = Prover::prove(&private, &public).unwrap();
            black_box(Verifier::verify(&proof).unwrap())
        })
    });
}

criterion_group!(
    benches,
    bench_zk_prove,
    bench_zk_verify,
    bench_zk_full_roundtrip,
);
criterion_main!(benches);
