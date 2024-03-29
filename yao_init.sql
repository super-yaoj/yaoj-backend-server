-- MySQL dump 10.13  Distrib 8.0.13, for Win64 (x86_64)
--
-- Host: localhost    Database: yaoj
-- ------------------------------------------------------
-- Server version	8.0.13

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
 SET NAMES utf8mb4 ;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `announcements`
--

DROP TABLE IF EXISTS `announcements`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE `announcements` (
  `blog_id` int(11) DEFAULT NULL,
  `release_time` datetime DEFAULT NULL,
  `priority` tinyint(4) DEFAULT NULL,
  `id` int(11) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `blog_id` (`blog_id`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `announcements`
--

LOCK TABLES `announcements` WRITE;
/*!40000 ALTER TABLE `announcements` DISABLE KEYS */;
/*!40000 ALTER TABLE `announcements` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `blog_comments`
--

DROP TABLE IF EXISTS `blog_comments`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE `blog_comments` (
  `blog_id` int(11) DEFAULT NULL,
  `create_time` datetime DEFAULT NULL,
  `author` int(11) DEFAULT NULL,
  `content` text,
  `like` int(11) DEFAULT NULL,
  `comment_id` int(11) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`comment_id`),
  KEY `blog_id` (`blog_id`),
  KEY `author` (`author`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `blog_comments`
--

LOCK TABLES `blog_comments` WRITE;
/*!40000 ALTER TABLE `blog_comments` DISABLE KEYS */;
/*!40000 ALTER TABLE `blog_comments` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `blogs`
--

DROP TABLE IF EXISTS `blogs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE `blogs` (
  `blog_id` int(11) NOT NULL AUTO_INCREMENT,
  `author` int(11) DEFAULT NULL,
  `title` varchar(100) DEFAULT NULL,
  `content` text,
  `private` tinyint(4) DEFAULT NULL,
  `create_time` datetime DEFAULT NULL,
  `comments` int(11) DEFAULT NULL,
  `like` int(11) DEFAULT NULL,
  PRIMARY KEY (`blog_id`),
  KEY `author` (`author`)
) ENGINE=MyISAM AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `blogs`
--

LOCK TABLES `blogs` WRITE;
/*!40000 ALTER TABLE `blogs` DISABLE KEYS */;
INSERT INTO `blogs` VALUES (1,1,'Yaoj 指南','### OJ 理念\n\n察今大部分 OJ 各有弊端，UOJ 虽开源但源码架构混乱不堪，运行效率低下；LOJ 采用前后端分离架构但渲染速度仍然不高。而同时，大部分 OJ 的造题理念均不够新颖——只支持传统造题模式，传统题、提答、交互题的配置虽比较简单，一旦涉及自由度更大、idea 更新颖的题目时便心有余而力不足，一道 UOJ 通信题往往耗费出题人半天到一天的时间琢磨 judger 源码并重写。洛谷等 OJ 更是完全不支持此等高级题目。\n\n故，为提高 OJ 效率，Yaoj 采用前后端 MVVM 架构和单页面模式，极大提高缓存复用率；同时后端采用 GO 编写，加之数据库精心设计，速度相比 UOJ 有了极大的提高。\n\n最重要的，黄队提出了题目的 Workflow 测试模式，使出题人们在不用花长时间琢磨 judger 源码的条件下可以高效率造出自己心满意足的新型题目。后文中也会介绍 Workflow 具体设计——通信题与普通题目的差别仅在于 Workflow 图中多了一个点而已。\n\n### OJ 功能\n\n#### 题目\n\n#### 比赛\n\n#### 杂项\n\n1. 左侧刷新按钮可实现页面内刷新（不需要重新加载 javascript），比普通刷新更流畅。\n2. 为了加快分页速度，所有分页的表格仅仅支持左右翻一页，但是你可以调整表格行数呀！\n3. 有一些表格翻页后页码会缓存（例如题目列表、比赛列表、用户列表，提交记录作为易变列表不具有缓存），第二次点击表格时会自动跳转到上次的页码，页面内刷新也无法恢复。但是如果使用 F5 刷新或点击分页表格的刷新按钮（最中间的那个）则会刷新数据且回到表首。\n4. 表格刷新按钮会清空查询信息和页码缓存并重新读取数据，如果你只是想重新读取数据而并不想清空查询信息，请点击页面左侧的刷新按钮。\n4. 右上角搜索框可以搜索用户名（输入用户名的任何一个前缀即可）。\n5. 无论是新建博客还是修改博客都会在本地产生额外的一份草稿（保存后草稿存在本地，不会随着关闭浏览器丢失，可以在 My blogs 中看到），而当点击提交时草稿会被删除并把内容发送到服务器。\n',0,'2022-07-04 00:39:00',0,0);
/*!40000 ALTER TABLE `blogs` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `click_like`
--

DROP TABLE IF EXISTS `click_like`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE `click_like` (
  `target` int(11) DEFAULT NULL,
  `id` int(11) DEFAULT NULL,
  `user_id` int(11) DEFAULT NULL,
  UNIQUE KEY `target` (`target`,`id`,`user_id`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `click_like`
--

LOCK TABLES `click_like` WRITE;
/*!40000 ALTER TABLE `click_like` DISABLE KEYS */;
/*!40000 ALTER TABLE `click_like` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `contest_participants`
--

DROP TABLE IF EXISTS `contest_participants`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE `contest_participants` (
  `contest_id` int(11) DEFAULT NULL,
  `user_id` int(11) DEFAULT NULL,
  UNIQUE KEY `contest_id` (`contest_id`,`user_id`),
  KEY `user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `contest_participants`
--

LOCK TABLES `contest_participants` WRITE;
/*!40000 ALTER TABLE `contest_participants` DISABLE KEYS */;
/*!40000 ALTER TABLE `contest_participants` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `contest_permissions`
--

DROP TABLE IF EXISTS `contest_permissions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE `contest_permissions` (
  `contest_id` int(11) DEFAULT NULL,
  `permission_id` int(11) DEFAULT NULL,
  UNIQUE KEY `contest_id` (`contest_id`,`permission_id`),
  KEY `permission_id` (`permission_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `contest_permissions`
--

LOCK TABLES `contest_permissions` WRITE;
/*!40000 ALTER TABLE `contest_permissions` DISABLE KEYS */;
/*!40000 ALTER TABLE `contest_permissions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `contest_problems`
--

DROP TABLE IF EXISTS `contest_problems`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE `contest_problems` (
  `contest_id` int(11) DEFAULT NULL,
  `problem_id` int(11) DEFAULT NULL,
  UNIQUE KEY `contest_id` (`contest_id`,`problem_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `contest_problems`
--

LOCK TABLES `contest_problems` WRITE;
/*!40000 ALTER TABLE `contest_problems` DISABLE KEYS */;
/*!40000 ALTER TABLE `contest_problems` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `contest_standing`
--

DROP TABLE IF EXISTS `contest_standing`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE `contest_standing` (
  `contest_id` int(11) NOT NULL,
  `standing` mediumblob,
  PRIMARY KEY (`contest_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `contest_standing`
--

LOCK TABLES `contest_standing` WRITE;
/*!40000 ALTER TABLE `contest_standing` DISABLE KEYS */;
/*!40000 ALTER TABLE `contest_standing` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `contests`
--

DROP TABLE IF EXISTS `contests`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE `contests` (
  `contest_id` int(11) NOT NULL AUTO_INCREMENT,
  `title` varchar(200) DEFAULT NULL,
  `start_time` datetime DEFAULT NULL,
  `end_time` datetime DEFAULT NULL,
  `pretest` tinyint(1) DEFAULT NULL,
  `score_private` tinyint(1) DEFAULT NULL,
  `like` int(11) DEFAULT NULL,
  `finished` tinyint(1) DEFAULT NULL,
  `registrants` int(11) DEFAULT NULL,
  PRIMARY KEY (`contest_id`),
  KEY `end_time` (`end_time`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `contests`
--

LOCK TABLES `contests` WRITE;
/*!40000 ALTER TABLE `contests` DISABLE KEYS */;
/*!40000 ALTER TABLE `contests` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `custom_tests`
--

DROP TABLE IF EXISTS `custom_tests`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE `custom_tests` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `content` mediumblob,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `custom_tests`
--

LOCK TABLES `custom_tests` WRITE;
/*!40000 ALTER TABLE `custom_tests` DISABLE KEYS */;
/*!40000 ALTER TABLE `custom_tests` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `permissions`
--

DROP TABLE IF EXISTS `permissions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE `permissions` (
  `permission_id` int(11) NOT NULL AUTO_INCREMENT,
  `permission_name` varchar(200) DEFAULT NULL,
  `count` int(11) DEFAULT '0',
  PRIMARY KEY (`permission_id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `permissions`
--

LOCK TABLES `permissions` WRITE;
/*!40000 ALTER TABLE `permissions` DISABLE KEYS */;
INSERT INTO `permissions` VALUES (1,'Default Group',1);
/*!40000 ALTER TABLE `permissions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `problem_permissions`
--

DROP TABLE IF EXISTS `problem_permissions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE `problem_permissions` (
  `problem_id` int(11) DEFAULT NULL,
  `permission_id` int(11) DEFAULT NULL,
  UNIQUE KEY `problem_id` (`problem_id`,`permission_id`),
  KEY `permission_id` (`permission_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `problem_permissions`
--

LOCK TABLES `problem_permissions` WRITE;
/*!40000 ALTER TABLE `problem_permissions` DISABLE KEYS */;
/*!40000 ALTER TABLE `problem_permissions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `problems`
--

DROP TABLE IF EXISTS `problems`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE `problems` (
  `problem_id` int(11) NOT NULL AUTO_INCREMENT,
  `title` varchar(200) DEFAULT NULL,
  `like` int(11) DEFAULT NULL,
  `check_sum` char(64) DEFAULT NULL,
  `allow_down` varchar(256) DEFAULT NULL,
  PRIMARY KEY (`problem_id`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `problems`
--

LOCK TABLES `problems` WRITE;
/*!40000 ALTER TABLE `problems` DISABLE KEYS */;
/*!40000 ALTER TABLE `problems` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `ratings`
--

DROP TABLE IF EXISTS `ratings`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE `ratings` (
  `user_id` int(11) DEFAULT NULL,
  `rating` int(11) DEFAULT NULL,
  `contest_id` int(11) DEFAULT NULL,
  `time` datetime DEFAULT NULL,
  KEY `user_id` (`user_id`),
  KEY `contest_id` (`contest_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `ratings`
--

LOCK TABLES `ratings` WRITE;
/*!40000 ALTER TABLE `ratings` DISABLE KEYS */;
/*!40000 ALTER TABLE `ratings` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `submission_details`
--

DROP TABLE IF EXISTS `submission_details`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE `submission_details` (
  `submission_id` int(11) NOT NULL,
  `content` longblob,
  `content_preview` mediumblob,
  `result` mediumblob,
  `pretest_result` mediumblob,
  `extra_result` mediumblob,
  PRIMARY KEY (`submission_id`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `submission_details`
--

LOCK TABLES `submission_details` WRITE;
/*!40000 ALTER TABLE `submission_details` DISABLE KEYS */;
/*!40000 ALTER TABLE `submission_details` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `submissions`
--

DROP TABLE IF EXISTS `submissions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE `submissions` (
  `submission_id` int(11) NOT NULL AUTO_INCREMENT,
  `submitter` int(11) DEFAULT NULL,
  `problem_id` int(11) DEFAULT NULL,
  `contest_id` int(11) DEFAULT NULL,
  `status` int(11) DEFAULT NULL,
  `score` float DEFAULT NULL,
  `time` int(11) DEFAULT NULL,
  `memory` int(11) DEFAULT NULL,
  `language` int(11) DEFAULT NULL,
  `submit_time` datetime DEFAULT NULL,
  `sample_score` float DEFAULT NULL,
  `accepted` int(11) DEFAULT NULL,
  `uuid` bigint(20) DEFAULT NULL,
  `length` int(11) DEFAULT NULL,
  PRIMARY KEY (`submission_id`),
  KEY `contest_id` (`contest_id`),
  KEY `status` (`status`),
  KEY `submitter` (`submitter`,`accepted`),
  KEY `problem_id` (`problem_id`,`accepted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `submissions`
--

LOCK TABLES `submissions` WRITE;
/*!40000 ALTER TABLE `submissions` DISABLE KEYS */;
/*!40000 ALTER TABLE `submissions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `user_info`
--

DROP TABLE IF EXISTS `user_info`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE `user_info` (
  `user_id` int(11) NOT NULL AUTO_INCREMENT,
  `user_name` varchar(30) DEFAULT NULL,
  `password` varchar(64) DEFAULT NULL,
  `motto` varchar(400) DEFAULT NULL,
  `rating` int(11) DEFAULT NULL,
  `register_time` datetime DEFAULT NULL,
  `remember_token` varchar(200) DEFAULT '',
  `user_group` int(11) DEFAULT '1',
  `gender` tinyint(4) DEFAULT '0',
  `email` varchar(100) DEFAULT NULL,
  `organization` varchar(200) DEFAULT '',
  PRIMARY KEY (`user_id`),
  UNIQUE KEY `user_name` (`user_name`),
  UNIQUE KEY `rating` (`rating` DESC,`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `user_info`
--

LOCK TABLES `user_info` WRITE;
/*!40000 ALTER TABLE `user_info` DISABLE KEYS */;
INSERT INTO `user_info` VALUES (1,'root','BFBBCEEB2D6402DF09EF79FE161F8A86A5466B183835F86FAF96584A09D41154','123123',0,'2022-06-18 20:56:40','cMSBK6sLTqB59UvhsKXbJe87UjyMTB7b',3,1,'zeyuan-w21@mails.tsinghua.edu.cn','清华大学交叉信息研究院');
/*!40000 ALTER TABLE `user_info` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `user_permissions`
--

DROP TABLE IF EXISTS `user_permissions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE `user_permissions` (
  `user_id` int(11) DEFAULT NULL,
  `permission_id` int(11) DEFAULT NULL,
  UNIQUE KEY `user_id` (`user_id`,`permission_id`),
  KEY `permission_id` (`permission_id`)
) ENGINE=MyISAM DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `user_permissions`
--

LOCK TABLES `user_permissions` WRITE;
/*!40000 ALTER TABLE `user_permissions` DISABLE KEYS */;
INSERT INTO `user_permissions` VALUES (1,1);
/*!40000 ALTER TABLE `user_permissions` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2022-07-27 17:02:58
DROP TABLE IF EXISTS `contest_dashboard`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE `contest_dashboard` (
  `contest_id` int(11) DEFAULT NULL,
  `dashboard` varchar(400) DEFAULT NULL,
  `time` datetime DEFAULT NULL,
  `id` int(11) NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (`id`),
  KEY `contest_id` (`contest_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci