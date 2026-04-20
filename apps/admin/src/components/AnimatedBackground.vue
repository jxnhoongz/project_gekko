<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref } from 'vue';
import * as THREE from 'three';

const container = ref<HTMLDivElement | null>(null);
let frame = 0;
let renderer: THREE.WebGLRenderer | null = null;
let scene: THREE.Scene | null = null;
let camera: THREE.PerspectiveCamera | null = null;
let mesh: THREE.Mesh | null = null;
let originalZ: Float32Array | null = null;
let ro: ResizeObserver | null = null;
let reduced = false;

function resize() {
  if (!container.value || !renderer || !camera) return;
  const w = container.value.clientWidth;
  const h = container.value.clientHeight;
  renderer.setSize(w, h, false);
  camera.aspect = w / h;
  camera.updateProjectionMatrix();
}

function animate() {
  if (!scene || !camera || !renderer || !mesh || !originalZ) return;
  const g = mesh.geometry as THREE.PlaneGeometry;
  const pos = g.attributes.position as THREE.BufferAttribute;
  const t = performance.now() * 0.00025;
  for (let i = 0; i < pos.count; i++) {
    const x = pos.getX(i);
    const y = pos.getY(i);
    const z = Math.sin(x * 0.55 + t * 2.2) * 0.9 + Math.cos(y * 0.45 + t * 1.6) * 0.7;
    pos.setZ(i, originalZ[i] + z);
  }
  pos.needsUpdate = true;
  g.computeVertexNormals();

  mesh.rotation.z = Math.sin(t * 0.3) * 0.05;
  renderer.render(scene, camera);
  frame = requestAnimationFrame(animate);
}

onMounted(() => {
  if (!container.value) return;
  reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

  scene = new THREE.Scene();
  scene.background = new THREE.Color('#faf8f3');
  scene.fog = new THREE.Fog('#faf8f3', 10, 40);

  camera = new THREE.PerspectiveCamera(55, 1, 0.1, 100);
  camera.position.set(0, 0, 14);

  renderer = new THREE.WebGLRenderer({ antialias: true, alpha: false });
  renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
  container.value.appendChild(renderer.domElement);

  // Low-poly wave plane — faceted look via flatShading
  const geom = new THREE.PlaneGeometry(36, 24, 28, 18);
  geom.rotateX(-Math.PI / 2.8);
  originalZ = new Float32Array(geom.attributes.position.count);
  const pos = geom.attributes.position;
  for (let i = 0; i < pos.count; i++) originalZ[i] = pos.getZ(i);

  const mat = new THREE.MeshStandardMaterial({
    color: 0xefc262,
    flatShading: true,
    metalness: 0.05,
    roughness: 0.85,
  });
  mesh = new THREE.Mesh(geom, mat);
  mesh.position.y = -3;
  scene.add(mesh);

  // Warm light from top-right per design rules
  const key = new THREE.DirectionalLight(0xffd79a, 1.1);
  key.position.set(8, 12, 6);
  scene.add(key);
  const fill = new THREE.AmbientLight(0xfaf8f3, 0.55);
  scene.add(fill);
  const rim = new THREE.PointLight(0xb06c12, 0.6, 40);
  rim.position.set(-6, -4, 8);
  scene.add(rim);

  resize();
  ro = new ResizeObserver(resize);
  ro.observe(container.value);

  if (reduced) {
    renderer.render(scene, camera);
  } else {
    frame = requestAnimationFrame(animate);
  }
});

onBeforeUnmount(() => {
  cancelAnimationFrame(frame);
  ro?.disconnect();
  if (mesh) {
    mesh.geometry.dispose();
    (mesh.material as THREE.Material).dispose();
  }
  renderer?.dispose();
  if (renderer && container.value?.contains(renderer.domElement)) {
    container.value.removeChild(renderer.domElement);
  }
});
</script>

<template>
  <div ref="container" class="absolute inset-0 -z-10" aria-hidden="true" />
</template>
