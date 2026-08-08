# --- build ---
FROM node:22-alpine AS build

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .

# URL бэка прокидывается на этапе сборки (Vite вшивает VITE_* в бандл)
ARG VITE_API_URL=http://localhost:8080
ARG VITE_JWT_SECRET
ENV VITE_API_URL=$VITE_API_URL
ENV VITE_JWT_SECRET=$VITE_JWT_SECRET

RUN npm run build

# --- runtime ---
FROM nginx:1.27-alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
