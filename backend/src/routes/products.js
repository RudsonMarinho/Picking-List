const express = require("express");
const prisma = require("../lib/prisma");
const { authRequired } = require("../middleware/auth");

const router = express.Router();

router.use(authRequired);

router.get("/", async (req, res) => {
  const products = await prisma.product.findMany({ orderBy: { name: "asc" } });
  res.json(products);
});

router.post("/", async (req, res) => {
  const { sku, name, unit } = req.body;

  if (!sku || !name) {
    return res.status(400).json({ error: "sku e name são obrigatórios" });
  }

  const existing = await prisma.product.findUnique({ where: { sku } });
  if (existing) {
    return res.status(409).json({ error: "Já existe um produto com este SKU" });
  }

  const product = await prisma.product.create({ data: { sku, name, unit: unit || "UN" } });
  res.status(201).json(product);
});

router.put("/:id", async (req, res) => {
  const { name, unit } = req.body;

  const product = await prisma.product.update({
    where: { id: req.params.id },
    data: { name, unit },
  });

  res.json(product);
});

router.delete("/:id", async (req, res) => {
  await prisma.product.delete({ where: { id: req.params.id } });
  res.status(204).send();
});

module.exports = router;
