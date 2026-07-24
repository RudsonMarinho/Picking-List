const { PrismaClient } = require("@prisma/client");
const bcrypt = require("bcryptjs");

const prisma = new PrismaClient();

async function main() {
  const passwordHash = await bcrypt.hash("admin123", 10);

  await prisma.user.upsert({
    where: { email: "admin@pickinglist.com" },
    update: {},
    create: {
      name: "Administrador",
      email: "admin@pickinglist.com",
      passwordHash,
      role: "ADMIN",
    },
  });

  console.log("Seed concluído. Login: admin@pickinglist.com / admin123");
}

main()
  .catch((err) => {
    console.error(err);
    process.exit(1);
  })
  .finally(() => prisma.$disconnect());
