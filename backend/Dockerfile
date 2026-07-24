FROM node:20-alpine

WORKDIR /app

COPY package*.json ./
RUN npm install

COPY . .
RUN npx prisma generate

EXPOSE 3333

CMD ["sh", "-c", "npx prisma db push --skip-generate && node prisma/seed.js && npm run dev"]
