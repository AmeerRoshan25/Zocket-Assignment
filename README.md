# Zocket Assignment

**Product Management System with Asynchronous Image Processing**
This project is a backend system built in Golang for managing products, featuring:

RESTful APIs for creating, retrieving, and filtering products.
PostgreSQL for data storage, including support for processed image URLs.
Asynchronous Image Processing using RabbitMQ/Kafka for downloading, compressing, and storing images in S3.
Redis Caching for faster data retrieval and reduced database load.
Structured Logging with tools like logrus or zap for detailed request and task tracking.
Robust Error Handling with retry mechanisms and dead-letter queues for task failures.

**Objective**
Build a highly scalable backend system in Golang for a product management application. The system should implement architectural best practices, including asynchronous image processing, caching, structured logging, and robust error handling.

## 🎯 Features
**API Endpoints:**
POST /products: Create a product with details like user ID, name, description, price, and image URLs.
GET /products/:id: Retrieve product details by ID, including processed image results.
GET /products: List all products for a user, with optional filtering by price range or product name.

**Data Storage:**
Use PostgreSQL to store product and user data.
Include a column (compressed_product_images) for storing URLs of processed images.

**Asynchronous Image Processing:**
Queue image processing tasks with RabbitMQ or Kafka upon product creation.
A separate microservice processes images (download, compress, upload to storage like S3) and updates the database.
**Caching:**
Use Redis to cache responses for GET /products/:id to reduce database load.
Implement cache invalidation to reflect data updates in real time.
**Enhanced Logging:**
Use libraries like logrus or zap for structured logging of API requests, errors, and background tasks.

**Error Handling:**
Handle failures in API operations and asynchronous tasks with retries or dead-letter queues.
**Testing:**
Unit tests for all API endpoints and core functions.
Integration tests to validate the entire workflow (e.g., asynchronous tasks and caching).
Performance benchmarking for cached vs. uncached responses.

**System Architecture**
Modular Design: Separate modules for APIs, tasks, caching, and logging.
Scalable Infrastructure: Handle increased API traffic, distributed caching, and image processing.
Transactional Consistency: Ensure consistent data across the database, cache, and message queue, with failure recovery mechanisms.
## 🖼 Sample Images

![image](https://github.com/user-attachments/assets/a88e7dbe-8521-4372-8ae2-de9c0e4840fe)

![image](https://github.com/user-attachments/assets/b2d653e9-d4b4-4a20-9b1c-b8da54a9fc62)
![image](https://github.com/user-attachments/assets/45aa01a4-dd59-4c62-9faa-e786f69d1e99)
![image](https://github.com/user-attachments/assets/eeaa0c23-5821-4645-86b0-bfb90b5bdc4d)



