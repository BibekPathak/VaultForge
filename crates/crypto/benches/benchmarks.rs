//! Benchmark tests for VaultForge crypto primitives.
//!
//! Run with: cargo bench --package crypto

use criterion::{black_box, criterion_group, criterion_main, Criterion};
use vaultforge_crypto::{AES256GCM, Sha256, SimpleKdf, constant_time_eq, merkle_root};

fn bench_sha256(c: &mut Criterion) {
    let data_1k = vec![0xABu8; 1024];
    let data_64k = vec![0xCDu8; 65536];
    let data_1m = vec![0xEFu8; 1024 * 1024];

    c.bench_function("sha256_1kb", |b| {
        b.iter(|| black_box(Sha256::hash(black_box(&data_1k))))
    });

    c.bench_function("sha256_64kb", |b| {
        b.iter(|| black_box(Sha256::hash(black_box(&data_64k))))
    });

    c.bench_function("sha256_1mb", |b| {
        b.iter(|| black_box(Sha256::hash(black_box(&data_1m))))
    });
}

fn bench_aes_gcm(c: &mut Criterion) {
    let key = [0x42u8; 32];
    let plaintext_small = vec![0xAAu8; 64];
    let plaintext_medium = vec![0xBBu8; 1024];
    let plaintext_large = vec![0xCCu8; 65536];

    c.bench_function("aes256gcm_encrypt_64b", |b| {
        b.iter(|| black_box(AES256GCM::encrypt(black_box(&key), plaintext_small.clone())))
    });

    c.bench_function("aes256gcm_encrypt_1kb", |b| {
        b.iter(|| black_box(AES256GCM::encrypt(black_box(&key), plaintext_medium.clone())))
    });

    c.bench_function("aes256gcm_encrypt_64kb", |b| {
        b.iter(|| black_box(AES256GCM::encrypt(black_box(&key), plaintext_large.clone())))
    });
}

fn bench_kdf(c: &mut Criterion) {
    let password = b"hunter2";
    let salt = [0x55u8; 16];

    c.bench_function("kdf_single_sha256_derive", |b| {
        b.iter(|| black_box(SimpleKdf::derive_key(black_box(password), &salt, 32)))
    });
}

fn bench_merkle(c: &mut Criterion) {
    let leaves_8: Vec<[u8; 32]> = (0u8..8).map(|i| Sha256::hash(&[i])).collect();
    let leaves_64: Vec<[u8; 32]> = (0u8..64).map(|i| Sha256::hash(&[i])).collect();
    let leaves_256: Vec<[u8; 32]> = (0u8..=255).map(|i| Sha256::hash(&[i])).collect();

    c.bench_function("merkle_root_8_leaves", |b| {
        b.iter(|| black_box(merkle_root(black_box(&leaves_8))))
    });

    c.bench_function("merkle_root_64_leaves", |b| {
        b.iter(|| black_box(merkle_root(black_box(&leaves_64))))
    });

    c.bench_function("merkle_root_256_leaves", |b| {
        b.iter(|| black_box(merkle_root(black_box(&leaves_256))))
    });
}

fn bench_constant_time_eq(c: &mut Criterion) {
    let a = [0xABu8; 32];
    let eq = [0xABu8; 32];
    let neq = [0xCDu8; 32];

    c.bench_function("constant_time_eq_equal", |b| {
        b.iter(|| black_box(constant_time_eq(&a, &eq)))
    });

    c.bench_function("constant_time_eq_not_equal", |b| {
        b.iter(|| black_box(constant_time_eq(&a, &neq)))
    });
}

criterion_group!(
    benches,
    bench_sha256,
    bench_aes_gcm,
    bench_kdf,
    bench_merkle,
    bench_constant_time_eq,
);
criterion_main!(benches);
