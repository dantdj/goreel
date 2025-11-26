FROM node:20-alpine

WORKDIR /app

# Copy package files first for caching
COPY package.json package-lock.json* ./

RUN npm install

COPY . .

EXPOSE 5173

# Run with --host to expose it to the network
CMD ["npm", "run", "dev", "--", "--host"]
